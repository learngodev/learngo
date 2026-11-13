package library

import (
	"context"
)

// ResourceRepository 管理教学资源。
type ResourceRepository interface {
	Create(ctx context.Context, resource *Resource) error
	Update(ctx context.Context, resource *Resource) error
	Delete(ctx context.Context, id, schoolID string) error
	GetByID(ctx context.Context, id, schoolID string) (*Resource, error)
	ListByTeacher(ctx context.Context, schoolID, teacherID string, departmentID string, page, size int) ([]Resource, int64, error)
	Browse(ctx context.Context, schoolID, query, departmentID, fileType, sort string, page, size int) ([]Resource, int64, error)
	IncrementViewCount(ctx context.Context, id string) error
	IncrementDownloadCount(ctx context.Context, id string) error
}

// ResourceFileRepository 管理资源与文件的关联。
type ResourceFileRepository interface {
	Create(ctx context.Context, link *ResourceFile) error
	Delete(ctx context.Context, resourceID, fileID string) error
	ListByResourceID(ctx context.Context, resourceID string) ([]ResourceFile, error)
	ValidateResourceFile(ctx context.Context, resourceID, fileID string) (bool, error)
}

// ResourceFavoriteRepository 管理资源收藏。
type ResourceFavoriteRepository interface {
	Create(ctx context.Context, favorite *ResourceFavorite) error
	Delete(ctx context.Context, resourceID, accountID string) error
	Exists(ctx context.Context, resourceID, accountID string) (bool, error)
	ListByAccount(ctx context.Context, accountID string, page, size int) ([]string, int64, error)
	BatchCheckFavorited(ctx context.Context, resourceIDs []string, accountID string) (map[string]bool, error)
	CountByResourceID(ctx context.Context, resourceID string) (int64, error)
}
