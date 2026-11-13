package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/domain/library"
	"learn-go/internal/domain/organization"
	"learn-go/internal/domain/roster"
	"learn-go/internal/domain/storage"

	"github.com/google/uuid"
)

var (
	ErrResourceNotFound     = errors.New("resource not found")
	ErrResourceAccessDenied = errors.New("resource access denied")
	ErrInvalidFileID        = errors.New("invalid file id")
)

// ResourceUsecase 负责教学资源的业务逻辑。
type ResourceUsecase struct {
	resources     library.ResourceRepository
	resourceFiles library.ResourceFileRepository
	favorites     library.ResourceFavoriteRepository
	teachers      roster.TeacherRepository
	fileUsecase   *FileUsecase
	departments   organization.DepartmentRepository
	files         storage.FileRepository
}

// NewResourceUsecase 创建 ResourceUsecase。
func NewResourceUsecase(
	resources library.ResourceRepository,
	resourceFiles library.ResourceFileRepository,
	favorites library.ResourceFavoriteRepository,
	teachers roster.TeacherRepository,
	fileUsecase *FileUsecase,
	departments organization.DepartmentRepository,
	files storage.FileRepository,
) *ResourceUsecase {
	return &ResourceUsecase{
		resources:     resources,
		resourceFiles: resourceFiles,
		favorites:     favorites,
		teachers:      teachers,
		fileUsecase:   fileUsecase,
		departments:   departments,
		files:         files,
	}
}

// CreateResourceRequest 表示创建资源的参数。
type CreateResourceRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	DepartmentID *string  `json:"department_id"`
	Tags         []string `json:"tags"`
	FileIDs      []string `json:"file_ids"`
}

// UpdateResourceRequest 表示更新资源的参数。
type UpdateResourceRequest struct {
	Title        *string  `json:"title"`
	Description  *string  `json:"description"`
	DepartmentID *string  `json:"department_id"`
	Tags         []string `json:"tags"`
}

// ResourceListParams 表示列表查询参数。
type ResourceListParams struct {
	DepartmentID  string
	FavoritedOnly bool
	Page          int
	Size          int
}

// BrowseResourcesParams 表示浏览查询参数。
type BrowseResourcesParams struct {
	Query           string
	DepartmentID    string
	FileType        string
	Sort            string
	FavoritedOnly   bool
	MyResourcesOnly bool // 仅显示当前用户的资源
	Page            int
	Size            int
}

// CreateResource 创建新资源。
func (s *ResourceUsecase) CreateResource(
	ctx context.Context,
	accountID, schoolID string,
	req CreateResourceRequest,
) (*library.Resource, error) {
	// 获取教师档案
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil || teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	// 校验学校是否一致
	if teacher.SchoolID != schoolID {
		return nil, ErrResourceAccessDenied
	}

	// 校验所有文件 ID 都属于该教师
	for _, fileID := range req.FileIDs {
		if _, err := s.files.GetByUploader(ctx, fileID, schoolID, accountID); err != nil {
			return nil, ErrInvalidFileID
		}
	}

	// 将标签序列化为 JSON
	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return nil, err
	}

	// 创建资源
	resource := &library.Resource{
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

	// 创建文件关联
	for i, fileID := range req.FileIDs {
		link := &library.ResourceFile{
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

// UpdateResource 更新资源。
func (s *ResourceUsecase) UpdateResource(
	ctx context.Context,
	resourceID, accountID, schoolID string,
	req UpdateResourceRequest,
) (*library.Resource, error) {
	// 校验归属
	resource, err := s.ensureTeacherResourceAccess(ctx, resourceID, accountID, schoolID)
	if err != nil {
		return nil, err
	}

	// 更新字段
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

// DeleteResource 删除资源。
func (s *ResourceUsecase) DeleteResource(
	ctx context.Context,
	resourceID, accountID, schoolID string,
) error {
	// 校验归属
	if _, err := s.ensureTeacherResourceAccess(ctx, resourceID, accountID, schoolID); err != nil {
		return err
	}

	return s.resources.Delete(ctx, resourceID, schoolID)
}

// ListTeacherResources 列出某教师创建的资源。
func (s *ResourceUsecase) ListTeacherResources(
	ctx context.Context,
	accountID, schoolID string,
	params ResourceListParams,
) ([]library.Resource, int64, error) {
	// 获取教师档案
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

	// 批量检查收藏状态
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

	// 按需仅显示已收藏的资源
	if params.FavoritedOnly {
		filtered := make([]library.Resource, 0)
		for _, r := range resources {
			if r.IsFavorited {
				filtered = append(filtered, r)
			}
		}
		return filtered, int64(len(filtered)), nil
	}

	return resources, total, nil
}

// BrowseResources 按搜索与筛选条件列出资源。
func (s *ResourceUsecase) BrowseResources(
	ctx context.Context,
	accountID, schoolID string,
	params BrowseResourcesParams,
) ([]library.Resource, int64, error) {
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

	// 仅看我的资源时获取教师 ID
	var myTeacherID string
	if params.MyResourcesOnly {
		teacher, err := s.teachers.GetByAccountID(ctx, accountID)
		if err == nil && teacher != nil {
			myTeacherID = teacher.ID
		}
	}

	// 为资源补充附加数据
	if len(resources) > 0 {
		resourceIDs := make([]string, len(resources))
		for i, r := range resources {
			resourceIDs[i] = r.ID
		}

		// 批量检查收藏状态
		favoriteMap, err := s.favorites.BatchCheckFavorited(ctx, resourceIDs, accountID)
		if err == nil {
			for i := range resources {
				resources[i].IsFavorited = favoriteMap[resources[i].ID]
			}
		}

		// 批量获取文件数
		for i := range resources {
			files, err := s.resourceFiles.ListByResourceID(ctx, resources[i].ID)
			if err == nil {
				resources[i].FileCount = len(files)
			}
		}

		// 批量获取收藏数
		for i := range resources {
			count, err := s.favorites.CountByResourceID(ctx, resources[i].ID)
			if err == nil {
				resources[i].FavoriteCount = int(count)
			}
		}
	}

	// 在内存中应用筛选条件
	filtered := make([]library.Resource, 0)
	for _, r := range resources {
		// 按收藏筛选
		if params.FavoritedOnly && !r.IsFavorited {
			continue
		}
		// 按我的资源筛选
		if params.MyResourcesOnly && myTeacherID != "" && r.TeacherID != myTeacherID {
			continue
		}
		filtered = append(filtered, r)
	}

	return filtered, int64(len(filtered)), nil
}

// GetResourceDetail 获取资源详情及其文件。
func (s *ResourceUsecase) GetResourceDetail(
	ctx context.Context,
	resourceID, accountID, schoolID string,
) (*library.Resource, []storage.File, error) {
	// 获取资源
	resource, err := s.resources.GetByID(ctx, resourceID, schoolID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, nil, ErrResourceNotFound
		}
		return nil, nil, err
	}

	// 异步增加浏览次数
	go func() {
		_ = s.resources.IncrementViewCount(context.Background(), resourceID)
	}()

	// 检查收藏状态
	isFavorited, _ := s.favorites.Exists(ctx, resourceID, accountID)
	resource.IsFavorited = isFavorited

	// 获取文件
	links, err := s.resourceFiles.ListByResourceID(ctx, resourceID)
	if err != nil {
		return nil, nil, err
	}

	fileIDs := make([]string, len(links))
	for i, link := range links {
		fileIDs[i] = link.FileID
	}

	var files []storage.File
	if len(fileIDs) > 0 {
		files, err = s.files.ListByIDs(ctx, fileIDs)
		if err != nil {
			return nil, nil, err
		}
	}

	return resource, files, nil
}

// AddFileToResource 向资源添加文件。
func (s *ResourceUsecase) AddFileToResource(
	ctx context.Context,
	resourceID, fileID, accountID, schoolID string,
) error {
	// 校验归属
	if _, err := s.ensureTeacherResourceAccess(ctx, resourceID, accountID, schoolID); err != nil {
		return err
	}

	// 校验文件归属
	if _, err := s.files.GetByUploader(ctx, fileID, schoolID, accountID); err != nil {
		return ErrInvalidFileID
	}

	// 获取当前最大排序序号
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

	// 创建关联
	link := &library.ResourceFile{
		ID:         uuid.NewString(),
		ResourceID: resourceID,
		FileID:     fileID,
		OrderIndex: maxOrder + 1,
		CreatedAt:  time.Now(),
	}

	return s.resourceFiles.Create(ctx, link)
}

// RemoveFileFromResource 从资源中移除文件。
func (s *ResourceUsecase) RemoveFileFromResource(
	ctx context.Context,
	resourceID, fileID, accountID, schoolID string,
) error {
	// 校验归属
	if _, err := s.ensureTeacherResourceAccess(ctx, resourceID, accountID, schoolID); err != nil {
		return err
	}

	return s.resourceFiles.Delete(ctx, resourceID, fileID)
}

// ToggleFavorite 切换收藏状态。
func (s *ResourceUsecase) ToggleFavorite(
	ctx context.Context,
	resourceID, accountID, schoolID string,
) (bool, error) {
	// 校验资源存在
	_, err := s.resources.GetByID(ctx, resourceID, schoolID)
	if err != nil {
		if err == domain.ErrNotFound {
			return false, ErrResourceNotFound
		}
		return false, err
	}

	// 检查是否已收藏
	exists, err := s.favorites.Exists(ctx, resourceID, accountID)
	if err != nil {
		return false, err
	}

	if exists {
		// 取消收藏
		if err := s.favorites.Delete(ctx, resourceID, accountID); err != nil {
			return false, err
		}
		return false, nil
	} else {
		// 添加收藏
		favorite := &library.ResourceFavorite{
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

// ListFavorites 列出已收藏的资源。
func (s *ResourceUsecase) ListFavorites(
	ctx context.Context,
	accountID, schoolID string,
	page, size int,
) ([]library.Resource, int64, error) {
	// 获取已收藏的资源 ID
	resourceIDs, total, err := s.favorites.ListByAccount(ctx, accountID, page, size)
	if err != nil {
		return nil, 0, err
	}

	if len(resourceIDs) == 0 {
		return []library.Resource{}, 0, nil
	}

	// 获取资源
	var resources []library.Resource
	for _, id := range resourceIDs {
		resource, err := s.resources.GetByID(ctx, id, schoolID)
		if err == nil {
			resource.IsFavorited = true
			resources = append(resources, *resource)
		}
	}

	return resources, total, nil
}

// IncrementDownloadCount 增加下载次数。
func (s *ResourceUsecase) IncrementDownloadCount(ctx context.Context, resourceID string) error {
	return s.resources.IncrementDownloadCount(ctx, resourceID)
}

// ensureTeacherResourceAccess 校验教师是否拥有该资源。
func (s *ResourceUsecase) ensureTeacherResourceAccess(
	ctx context.Context,
	resourceID, accountID, schoolID string,
) (*library.Resource, error) {
	// 获取教师档案
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil || teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	// 获取资源
	resource, err := s.resources.GetByID(ctx, resourceID, schoolID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, ErrResourceNotFound
		}
		return nil, err
	}

	// 校验归属
	if resource.TeacherID != teacher.ID {
		return nil, ErrResourceAccessDenied
	}

	return resource, nil
}
