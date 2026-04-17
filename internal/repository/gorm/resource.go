package gormrepo

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// ResourceStore implements ResourceRepository using GORM.
type ResourceStore struct {
	db *gorm.DB
}

// NewResourceStore creates a new ResourceStore.
func NewResourceStore(db *gorm.DB) *ResourceStore {
	return &ResourceStore{db: db}
}

// Create creates a new resource.
func (r *ResourceStore) Create(ctx context.Context, resource *domain.Resource) error {
	return r.db.WithContext(ctx).Create(resource).Error
}

// Update updates a resource.
func (r *ResourceStore) Update(ctx context.Context, resource *domain.Resource) error {
	return r.db.WithContext(ctx).Save(resource).Error
}

// Delete soft deletes a resource.
func (r *ResourceStore) Delete(ctx context.Context, id, schoolID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", id, schoolID).
		Delete(&domain.Resource{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// GetByID retrieves a resource by ID.
func (r *ResourceStore) GetByID(ctx context.Context, id, schoolID string) (*domain.Resource, error) {
	var resource domain.Resource
	err := r.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", id, schoolID).
		First(&resource).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &resource, nil
}

// ListByTeacher lists resources created by a teacher.
func (r *ResourceStore) ListByTeacher(
	ctx context.Context,
	schoolID, teacherID string,
	departmentID string,
	page, size int,
) ([]domain.Resource, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Resource{}).
		Select("resources.*, accounts.display_name as teacher_name, departments.name as department_name").
		Joins("LEFT JOIN teachers ON teachers.id = resources.teacher_id").
		Joins("LEFT JOIN accounts ON accounts.id = teachers.account_id").
		Joins("LEFT JOIN departments ON departments.id = resources.department_id").
		Where("resources.school_id = ? AND resources.teacher_id = ?", schoolID, teacherID)

	if departmentID != "" {
		query = query.Where("resources.department_id = ?", departmentID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var resources []domain.Resource
	err := query.Order("resources.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&resources).Error

	return resources, total, err
}

// Browse lists resources with search and filters.
func (r *ResourceStore) Browse(
	ctx context.Context,
	schoolID, query, departmentID, fileType, sort string,
	page, size int,
) ([]domain.Resource, int64, error) {
	db := r.db.WithContext(ctx).Model(&domain.Resource{}).
		Select("resources.*, accounts.display_name as teacher_name, departments.name as department_name").
		Joins("LEFT JOIN teachers ON teachers.id = resources.teacher_id").
		Joins("LEFT JOIN accounts ON accounts.id = teachers.account_id").
		Joins("LEFT JOIN departments ON departments.id = resources.department_id").
		Where("resources.school_id = ?", schoolID)

	// Keyword search
	if query != "" {
		searchPattern := "%" + query + "%"
		db = db.Where("resources.title LIKE ? OR resources.description LIKE ?", searchPattern, searchPattern)
	}

	// Department filter
	if departmentID != "" {
		db = db.Where("resources.department_id = ?", departmentID)
	}

	// File type filter
	if fileType != "" {
		db = db.Joins("JOIN resource_files rf ON rf.resource_id = resources.id").
			Joins("JOIN files f ON f.id = rf.file_id").
			Where("f.type LIKE ?", fileType+"%").
			Distinct()
	}

	// Count total
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sorting
	switch sort {
	case "popular":
		db = db.Order("resources.view_count DESC")
	case "downloads":
		db = db.Order("resources.download_count DESC")
	default:
		db = db.Order("resources.created_at DESC")
	}

	// Pagination
	var resources []domain.Resource
	err := db.Offset((page - 1) * size).
		Limit(size).
		Find(&resources).Error

	return resources, total, err
}

// IncrementViewCount increments the view count.
func (r *ResourceStore) IncrementViewCount(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Resource{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).
		Error
}

// IncrementDownloadCount increments the download count.
func (r *ResourceStore) IncrementDownloadCount(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Resource{}).
		Where("id = ?", id).
		UpdateColumn("download_count", gorm.Expr("download_count + 1")).
		Error
}

// ResourceFileStore implements ResourceFileRepository using GORM.
type ResourceFileStore struct {
	db *gorm.DB
}

// NewResourceFileStore creates a new ResourceFileStore.
func NewResourceFileStore(db *gorm.DB) *ResourceFileStore {
	return &ResourceFileStore{db: db}
}

// Create creates a resource-file link.
func (r *ResourceFileStore) Create(ctx context.Context, link *domain.ResourceFile) error {
	return r.db.WithContext(ctx).Create(link).Error
}

// Delete removes a resource-file link.
func (r *ResourceFileStore) Delete(ctx context.Context, resourceID, fileID string) error {
	result := r.db.WithContext(ctx).
		Where("resource_id = ? AND file_id = ?", resourceID, fileID).
		Delete(&domain.ResourceFile{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// ListByResourceID lists files for a resource.
func (r *ResourceFileStore) ListByResourceID(ctx context.Context, resourceID string) ([]domain.ResourceFile, error) {
	var links []domain.ResourceFile
	err := r.db.WithContext(ctx).
		Where("resource_id = ?", resourceID).
		Order("order_index ASC").
		Find(&links).Error
	return links, err
}

// ValidateResourceFile checks if a file belongs to a resource.
func (r *ResourceFileStore) ValidateResourceFile(ctx context.Context, resourceID, fileID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.ResourceFile{}).
		Where("resource_id = ? AND file_id = ?", resourceID, fileID).
		Count(&count).Error
	return count > 0, err
}

// ResourceFavoriteStore implements ResourceFavoriteRepository using GORM.
type ResourceFavoriteStore struct {
	db *gorm.DB
}

// NewResourceFavoriteStore creates a new ResourceFavoriteStore.
func NewResourceFavoriteStore(db *gorm.DB) *ResourceFavoriteStore {
	return &ResourceFavoriteStore{db: db}
}

// Create creates a favorite.
func (r *ResourceFavoriteStore) Create(ctx context.Context, favorite *domain.ResourceFavorite) error {
	return r.db.WithContext(ctx).Create(favorite).Error
}

// Delete removes a favorite.
func (r *ResourceFavoriteStore) Delete(ctx context.Context, resourceID, accountID string) error {
	result := r.db.WithContext(ctx).
		Where("resource_id = ? AND account_id = ?", resourceID, accountID).
		Delete(&domain.ResourceFavorite{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// Exists checks if a favorite exists.
func (r *ResourceFavoriteStore) Exists(ctx context.Context, resourceID, accountID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.ResourceFavorite{}).
		Where("resource_id = ? AND account_id = ?", resourceID, accountID).
		Count(&count).Error
	return count > 0, err
}

// ListByAccount lists favorited resource IDs for an account.
func (r *ResourceFavoriteStore) ListByAccount(ctx context.Context, accountID string, page, size int) ([]string, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&domain.ResourceFavorite{}).
		Where("account_id = ?", accountID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var favorites []domain.ResourceFavorite
	err := query.Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&favorites).Error
	if err != nil {
		return nil, 0, err
	}

	resourceIDs := make([]string, len(favorites))
	for i, f := range favorites {
		resourceIDs[i] = f.ResourceID
	}

	return resourceIDs, total, nil
}

// BatchCheckFavorited checks favorite status for multiple resources.
func (r *ResourceFavoriteStore) BatchCheckFavorited(ctx context.Context, resourceIDs []string, accountID string) (map[string]bool, error) {
	if len(resourceIDs) == 0 {
		return make(map[string]bool), nil
	}

	var favorites []domain.ResourceFavorite
	err := r.db.WithContext(ctx).
		Where("resource_id IN ? AND account_id = ?", resourceIDs, accountID).
		Find(&favorites).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for _, id := range resourceIDs {
		result[id] = false
	}
	for _, f := range favorites {
		result[f.ResourceID] = true
	}

	return result, nil
}

// CountByResourceID counts favorites for a resource.
func (r *ResourceFavoriteStore) CountByResourceID(ctx context.Context, resourceID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.ResourceFavorite{}).
		Where("resource_id = ?", resourceID).
		Count(&count).Error
	return count, err
}
