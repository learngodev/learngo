package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrResourceNotFound     = errors.New("resource not found")
	ErrResourceAccessDenied = errors.New("resource access denied")
	ErrInvalidFileID        = errors.New("invalid file id")
)

// ResourceService handles resource business logic.
type ResourceService struct {
	resources     repository.ResourceRepository
	resourceFiles repository.ResourceFileRepository
	favorites     repository.ResourceFavoriteRepository
	teachers      repository.TeacherRepository
	fileService   *FileService
	departments   repository.DepartmentRepository
	db            *gorm.DB
}

// NewResourceService creates a new ResourceService.
func NewResourceService(
	resources repository.ResourceRepository,
	resourceFiles repository.ResourceFileRepository,
	favorites repository.ResourceFavoriteRepository,
	teachers repository.TeacherRepository,
	fileService *FileService,
	departments repository.DepartmentRepository,
	db *gorm.DB,
) *ResourceService {
	return &ResourceService{
		resources:     resources,
		resourceFiles: resourceFiles,
		favorites:     favorites,
		teachers:      teachers,
		fileService:   fileService,
		departments:   departments,
		db:            db,
	}
}

// CreateResourceRequest represents resource creation parameters.
type CreateResourceRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	DepartmentID *string  `json:"department_id"`
	Tags         []string `json:"tags"`
	FileIDs      []string `json:"file_ids"`
}

// UpdateResourceRequest represents resource update parameters.
type UpdateResourceRequest struct {
	Title        *string  `json:"title"`
	Description  *string  `json:"description"`
	DepartmentID *string  `json:"department_id"`
	Tags         []string `json:"tags"`
}

// ResourceListParams represents list parameters.
type ResourceListParams struct {
	DepartmentID  string
	FavoritedOnly bool
	Page          int
	Size          int
}

// BrowseResourcesParams represents browse parameters.
type BrowseResourcesParams struct {
	Query           string
	DepartmentID    string
	FileType        string
	Sort            string
	FavoritedOnly   bool
	MyResourcesOnly bool // Filter by current user's resources
	Page            int
	Size            int
}

// CreateResource creates a new resource.
func (s *ResourceService) CreateResource(
	ctx context.Context,
	accountID, schoolID string,
	req CreateResourceRequest,
) (*domain.Resource, error) {
	// Get teacher profile
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil || teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	// Verify school match
	if teacher.SchoolID != schoolID {
		return nil, ErrResourceAccessDenied
	}

	// Validate all file IDs belong to this teacher
	for _, fileID := range req.FileIDs {
		var file domain.File
		err := s.db.WithContext(ctx).Where("id = ? AND uploader_id = ? AND school_id = ?", fileID, accountID, schoolID).First(&file).Error
		if err != nil {
			return nil, ErrInvalidFileID
		}
	}

	// Marshal tags to JSON
	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return nil, err
	}

	// Create resource
	resource := &domain.Resource{
		ID:           uuid.NewString(),
		SchoolID:     schoolID,
		TeacherID:    teacher.ID,
		DepartmentID: req.DepartmentID,
		Title:        req.Title,
		Description:  req.Description,
		Tags:         string(tagsJSON),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.resources.Create(ctx, resource); err != nil {
		return nil, err
	}

	// Create file associations
	for i, fileID := range req.FileIDs {
		link := &domain.ResourceFile{
			ID:         uuid.NewString(),
			ResourceID: resource.ID,
			FileID:     fileID,
			OrderIndex: i,
			CreatedAt:  time.Now(),
		}
		if err := s.resourceFiles.Create(ctx, link); err != nil {
			return nil, err
		}
	}

	return resource, nil
}

// UpdateResource updates a resource.
func (s *ResourceService) UpdateResource(
	ctx context.Context,
	resourceID, accountID, schoolID string,
	req UpdateResourceRequest,
) (*domain.Resource, error) {
	// Verify ownership
	resource, err := s.ensureTeacherResourceAccess(ctx, resourceID, accountID, schoolID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Title != nil {
		resource.Title = *req.Title
	}
	if req.Description != nil {
		resource.Description = *req.Description
	}
	if req.DepartmentID != nil {
		resource.DepartmentID = req.DepartmentID
	}
	if req.Tags != nil {
		tagsJSON, err := json.Marshal(req.Tags)
		if err != nil {
			return nil, err
		}
		resource.Tags = string(tagsJSON)
	}

	resource.UpdatedAt = time.Now()

	if err := s.resources.Update(ctx, resource); err != nil {
		return nil, err
	}

	return resource, nil
}

// DeleteResource deletes a resource.
func (s *ResourceService) DeleteResource(
	ctx context.Context,
	resourceID, accountID, schoolID string,
) error {
	// Verify ownership
	if _, err := s.ensureTeacherResourceAccess(ctx, resourceID, accountID, schoolID); err != nil {
		return err
	}

	return s.resources.Delete(ctx, resourceID, schoolID)
}

// ListTeacherResources lists resources created by a teacher.
func (s *ResourceService) ListTeacherResources(
	ctx context.Context,
	accountID, schoolID string,
	params ResourceListParams,
) ([]domain.Resource, int64, error) {
	// Get teacher profile
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil || teacher == nil {
		return nil, 0, ErrTeacherProfileNotFound
	}

	resources, total, err := s.resources.ListByTeacher(
		ctx,
		schoolID,
		teacher.ID,
		params.DepartmentID,
		params.Page,
		params.Size,
	)
	if err != nil {
		return nil, 0, err
	}

	// Batch check favorite status
	if len(resources) > 0 {
		resourceIDs := make([]string, len(resources))
		for i, r := range resources {
			resourceIDs[i] = r.ID
		}

		favoriteMap, err := s.favorites.BatchCheckFavorited(ctx, resourceIDs, accountID)
		if err == nil {
			for i := range resources {
				resources[i].IsFavorited = favoriteMap[resources[i].ID]
			}
		}
	}

	// Filter by favorited if requested
	if params.FavoritedOnly {
		filtered := make([]domain.Resource, 0)
		for _, r := range resources {
			if r.IsFavorited {
				filtered = append(filtered, r)
			}
		}
		return filtered, int64(len(filtered)), nil
	}

	return resources, total, nil
}

// BrowseResources lists resources with search and filters.
func (s *ResourceService) BrowseResources(
	ctx context.Context,
	accountID, schoolID string,
	params BrowseResourcesParams,
) ([]domain.Resource, int64, error) {
	resources, _, err := s.resources.Browse(
		ctx,
		schoolID,
		params.Query,
		params.DepartmentID,
		params.FileType,
		params.Sort,
		params.Page,
		params.Size,
	)
	if err != nil {
		return nil, 0, err
	}

	// Get teacher ID if filtering by my resources
	var myTeacherID string
	if params.MyResourcesOnly {
		teacher, err := s.teachers.GetByAccountID(ctx, accountID)
		if err == nil && teacher != nil {
			myTeacherID = teacher.ID
		}
	}

	// Enrich resources with additional data
	if len(resources) > 0 {
		resourceIDs := make([]string, len(resources))
		for i, r := range resources {
			resourceIDs[i] = r.ID
		}

		// Batch check favorite status
		favoriteMap, err := s.favorites.BatchCheckFavorited(ctx, resourceIDs, accountID)
		if err == nil {
			for i := range resources {
				resources[i].IsFavorited = favoriteMap[resources[i].ID]
			}
		}

		// Batch get file counts
		for i := range resources {
			files, err := s.resourceFiles.ListByResourceID(ctx, resources[i].ID)
			if err == nil {
				resources[i].FileCount = len(files)
			}
		}

		// Batch get favorite counts
		for i := range resources {
			count, err := s.favorites.CountByResourceID(ctx, resources[i].ID)
			if err == nil {
				resources[i].FavoriteCount = int(count)
			}
		}
	}

	// Apply client-side filters
	filtered := make([]domain.Resource, 0)
	for _, r := range resources {
		// Filter by favorited
		if params.FavoritedOnly && !r.IsFavorited {
			continue
		}
		// Filter by my resources
		if params.MyResourcesOnly && myTeacherID != "" && r.TeacherID != myTeacherID {
			continue
		}
		filtered = append(filtered, r)
	}

	return filtered, int64(len(filtered)), nil
}

// GetResourceDetail gets resource detail with files.
func (s *ResourceService) GetResourceDetail(
	ctx context.Context,
	resourceID, accountID, schoolID string,
) (*domain.Resource, []domain.File, error) {
	// Get resource
	resource, err := s.resources.GetByID(ctx, resourceID, schoolID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil, ErrResourceNotFound
		}
		return nil, nil, err
	}

	// Increment view count asynchronously
	go func() {
		_ = s.resources.IncrementViewCount(context.Background(), resourceID)
	}()

	// Check favorite status
	isFavorited, _ := s.favorites.Exists(ctx, resourceID, accountID)
	resource.IsFavorited = isFavorited

	// Get files
	links, err := s.resourceFiles.ListByResourceID(ctx, resourceID)
	if err != nil {
		return nil, nil, err
	}

	fileIDs := make([]string, len(links))
	for i, link := range links {
		fileIDs[i] = link.FileID
	}

	var files []domain.File
	if len(fileIDs) > 0 {
		err = s.db.WithContext(ctx).Where("id IN ?", fileIDs).Find(&files).Error
		if err != nil {
			return nil, nil, err
		}
	}

	return resource, files, nil
}

// AddFileToResource adds a file to a resource.
func (s *ResourceService) AddFileToResource(
	ctx context.Context,
	resourceID, fileID, accountID, schoolID string,
) error {
	// Verify ownership
	if _, err := s.ensureTeacherResourceAccess(ctx, resourceID, accountID, schoolID); err != nil {
		return err
	}

	// Verify file ownership
	var file domain.File
	err := s.db.WithContext(ctx).Where("id = ? AND uploader_id = ? AND school_id = ?", fileID, accountID, schoolID).First(&file).Error
	if err != nil {
		return ErrInvalidFileID
	}

	// Get current max order index
	links, err := s.resourceFiles.ListByResourceID(ctx, resourceID)
	if err != nil {
		return err
	}

	maxOrder := 0
	for _, link := range links {
		if link.OrderIndex > maxOrder {
			maxOrder = link.OrderIndex
		}
	}

	// Create link
	link := &domain.ResourceFile{
		ID:         uuid.NewString(),
		ResourceID: resourceID,
		FileID:     fileID,
		OrderIndex: maxOrder + 1,
		CreatedAt:  time.Now(),
	}

	return s.resourceFiles.Create(ctx, link)
}

// RemoveFileFromResource removes a file from a resource.
func (s *ResourceService) RemoveFileFromResource(
	ctx context.Context,
	resourceID, fileID, accountID, schoolID string,
) error {
	// Verify ownership
	if _, err := s.ensureTeacherResourceAccess(ctx, resourceID, accountID, schoolID); err != nil {
		return err
	}

	return s.resourceFiles.Delete(ctx, resourceID, fileID)
}

// ToggleFavorite toggles favorite status.
func (s *ResourceService) ToggleFavorite(
	ctx context.Context,
	resourceID, accountID, schoolID string,
) (bool, error) {
	// Verify resource exists
	_, err := s.resources.GetByID(ctx, resourceID, schoolID)
	if err != nil {
		if err == repository.ErrNotFound {
			return false, ErrResourceNotFound
		}
		return false, err
	}

	// Check if already favorited
	exists, err := s.favorites.Exists(ctx, resourceID, accountID)
	if err != nil {
		return false, err
	}

	if exists {
		// Remove favorite
		if err := s.favorites.Delete(ctx, resourceID, accountID); err != nil {
			return false, err
		}
		return false, nil
	} else {
		// Add favorite
		favorite := &domain.ResourceFavorite{
			ID:         uuid.NewString(),
			ResourceID: resourceID,
			AccountID:  accountID,
			CreatedAt:  time.Now(),
		}
		if err := s.favorites.Create(ctx, favorite); err != nil {
			return false, err
		}
		return true, nil
	}
}

// ListFavorites lists favorited resources.
func (s *ResourceService) ListFavorites(
	ctx context.Context,
	accountID, schoolID string,
	page, size int,
) ([]domain.Resource, int64, error) {
	// Get favorited resource IDs
	resourceIDs, total, err := s.favorites.ListByAccount(ctx, accountID, page, size)
	if err != nil {
		return nil, 0, err
	}

	if len(resourceIDs) == 0 {
		return []domain.Resource{}, 0, nil
	}

	// Get resources
	var resources []domain.Resource
	for _, id := range resourceIDs {
		resource, err := s.resources.GetByID(ctx, id, schoolID)
		if err == nil {
			resource.IsFavorited = true
			resources = append(resources, *resource)
		}
	}

	return resources, total, nil
}

// IncrementDownloadCount increments download count.
func (s *ResourceService) IncrementDownloadCount(ctx context.Context, resourceID string) error {
	return s.resources.IncrementDownloadCount(ctx, resourceID)
}

// ensureTeacherResourceAccess verifies teacher owns the resource.
func (s *ResourceService) ensureTeacherResourceAccess(
	ctx context.Context,
	resourceID, accountID, schoolID string,
) (*domain.Resource, error) {
	// Get teacher profile
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil || teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	// Get resource
	resource, err := s.resources.GetByID(ctx, resourceID, schoolID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}

	// Verify ownership
	if resource.TeacherID != teacher.ID {
		return nil, ErrResourceAccessDenied
	}

	return resource, nil
}
