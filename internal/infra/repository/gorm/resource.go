package gormrepo

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/domain/library"

	"gorm.io/gorm"
)

// ResourceStore 基于 GORM 实现 library.ResourceRepository。
type ResourceStore struct {
	db *gorm.DB
}

// NewResourceStore 创建 ResourceStore。
func NewResourceStore(db *gorm.DB) *ResourceStore {
	return &ResourceStore{db: db}
}

// Create 创建新资源。
func (r *ResourceStore) Create(ctx context.Context, resource *library.Resource) error {
	return r.db.WithContext(ctx).Create(resource).Error
}

// Update 更新资源。
func (r *ResourceStore) Update(ctx context.Context, resource *library.Resource) error {
	return r.db.WithContext(ctx).Save(resource).Error
}

// Delete 软删除资源。
func (r *ResourceStore) Delete(ctx context.Context, id, schoolID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", id, schoolID).
		Delete(&library.Resource{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetByID 按 ID 查询资源。
func (r *ResourceStore) GetByID(ctx context.Context, id, schoolID string) (*library.Resource, error) {
	var resource library.Resource
	err := r.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", id, schoolID).
		First(&resource).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &resource, nil
}

// ListByTeacher 列出某教师创建的资源。
func (r *ResourceStore) ListByTeacher(
	ctx context.Context,
	schoolID, teacherID string,
	departmentID string,
	page, size int,
) ([]library.Resource, int64, error) {
	query := r.db.WithContext(ctx).Model(&library.Resource{}).
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

	var resources []library.Resource
	err := query.Order("resources.created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&resources).Error

	return resources, total, err
}

// Browse 按搜索与筛选条件列出资源。
func (r *ResourceStore) Browse(
	ctx context.Context,
	schoolID, query, departmentID, fileType, sort string,
	page, size int,
) ([]library.Resource, int64, error) {
	db := r.db.WithContext(ctx).Model(&library.Resource{}).
		Select("resources.*, accounts.display_name as teacher_name, departments.name as department_name").
		Joins("LEFT JOIN teachers ON teachers.id = resources.teacher_id").
		Joins("LEFT JOIN accounts ON accounts.id = teachers.account_id").
		Joins("LEFT JOIN departments ON departments.id = resources.department_id").
		Where("resources.school_id = ?", schoolID)

	// 关键字搜索
	if query != "" {
		searchPattern := "%" + query + "%"
		db = db.Where("resources.title LIKE ? OR resources.description LIKE ?", searchPattern, searchPattern)
	}

	// 按院系筛选
	if departmentID != "" {
		db = db.Where("resources.department_id = ?", departmentID)
	}

	// 按文件类型筛选
	if fileType != "" {
		db = db.Joins("JOIN resource_files rf ON rf.resource_id = resources.id").
			Joins("JOIN files f ON f.id = rf.file_id").
			Where("f.type LIKE ?", fileType+"%").
			Distinct()
	}

	// 统计总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	switch sort {
	case "popular":
		db = db.Order("resources.view_count DESC")
	case "downloads":
		db = db.Order("resources.download_count DESC")
	default:
		db = db.Order("resources.created_at DESC")
	}

	// 分页
	var resources []library.Resource
	err := db.Offset((page - 1) * size).
		Limit(size).
		Find(&resources).Error

	return resources, total, err
}

// IncrementViewCount 增加浏览次数。
func (r *ResourceStore) IncrementViewCount(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&library.Resource{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).
		Error
}

// IncrementDownloadCount 增加下载次数。
func (r *ResourceStore) IncrementDownloadCount(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&library.Resource{}).
		Where("id = ?", id).
		UpdateColumn("download_count", gorm.Expr("download_count + 1")).
		Error
}

// ResourceFileStore 基于 GORM 实现 library.ResourceFileRepository。
type ResourceFileStore struct {
	db *gorm.DB
}

// NewResourceFileStore 创建 ResourceFileStore。
func NewResourceFileStore(db *gorm.DB) *ResourceFileStore {
	return &ResourceFileStore{db: db}
}

// Create 创建资源与文件的关联。
func (r *ResourceFileStore) Create(ctx context.Context, link *library.ResourceFile) error {
	return r.db.WithContext(ctx).Create(link).Error
}

// Delete 删除资源与文件的关联。
func (r *ResourceFileStore) Delete(ctx context.Context, resourceID, fileID string) error {
	result := r.db.WithContext(ctx).
		Where("resource_id = ? AND file_id = ?", resourceID, fileID).
		Delete(&library.ResourceFile{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListByResourceID 列出某资源下的文件。
func (r *ResourceFileStore) ListByResourceID(ctx context.Context, resourceID string) ([]library.ResourceFile, error) {
	var links []library.ResourceFile
	err := r.db.WithContext(ctx).
		Where("resource_id = ?", resourceID).
		Order("order_index ASC").
		Find(&links).Error
	return links, err
}

// ValidateResourceFile 检查文件是否属于该资源。
func (r *ResourceFileStore) ValidateResourceFile(ctx context.Context, resourceID, fileID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&library.ResourceFile{}).
		Where("resource_id = ? AND file_id = ?", resourceID, fileID).
		Count(&count).Error
	return count > 0, err
}

// ResourceFavoriteStore 基于 GORM 实现 library.ResourceFavoriteRepository。
type ResourceFavoriteStore struct {
	db *gorm.DB
}

// NewResourceFavoriteStore 创建 ResourceFavoriteStore。
func NewResourceFavoriteStore(db *gorm.DB) *ResourceFavoriteStore {
	return &ResourceFavoriteStore{db: db}
}

// Create 新增收藏。
func (r *ResourceFavoriteStore) Create(ctx context.Context, favorite *library.ResourceFavorite) error {
	return r.db.WithContext(ctx).Create(favorite).Error
}

// Delete 取消收藏。
func (r *ResourceFavoriteStore) Delete(ctx context.Context, resourceID, accountID string) error {
	result := r.db.WithContext(ctx).
		Where("resource_id = ? AND account_id = ?", resourceID, accountID).
		Delete(&library.ResourceFavorite{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Exists 检查收藏是否存在。
func (r *ResourceFavoriteStore) Exists(ctx context.Context, resourceID, accountID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&library.ResourceFavorite{}).
		Where("resource_id = ? AND account_id = ?", resourceID, accountID).
		Count(&count).Error
	return count > 0, err
}

// ListByAccount 列出某账号收藏的资源 ID。
func (r *ResourceFavoriteStore) ListByAccount(ctx context.Context, accountID string, page, size int) ([]string, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&library.ResourceFavorite{}).
		Where("account_id = ?", accountID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var favorites []library.ResourceFavorite
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

// BatchCheckFavorited 批量检查多个资源的收藏状态。
func (r *ResourceFavoriteStore) BatchCheckFavorited(ctx context.Context, resourceIDs []string, accountID string) (map[string]bool, error) {
	if len(resourceIDs) == 0 {
		return make(map[string]bool), nil
	}

	var favorites []library.ResourceFavorite
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

// CountByResourceID 统计某资源的收藏数。
func (r *ResourceFavoriteStore) CountByResourceID(ctx context.Context, resourceID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&library.ResourceFavorite{}).
		Where("resource_id = ?", resourceID).
		Count(&count).Error
	return count, err
}

var (
	_ library.ResourceRepository         = (*ResourceStore)(nil)
	_ library.ResourceFileRepository     = (*ResourceFileStore)(nil)
	_ library.ResourceFavoriteRepository = (*ResourceFavoriteStore)(nil)
)
