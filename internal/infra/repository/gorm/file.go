package gormrepo

import (
	"context"

	"gorm.io/gorm"

	"learn-go/internal/domain/storage"
)

// FileStore 基于 GORM 实现 storage.FileRepository。
type FileStore struct {
	db *gorm.DB
}

// NewFileStore 创建 FileStore。
func NewFileStore(db *gorm.DB) *FileStore {
	return &FileStore{db: db}
}

func (s *FileStore) Create(ctx context.Context, file *storage.File) error {
	return s.db.WithContext(ctx).Create(file).Error
}

func (s *FileStore) GetByID(ctx context.Context, id, schoolID string) (*storage.File, error) {
	var file storage.File
	if err := s.db.WithContext(ctx).First(&file, "id = ? AND school_id = ?", id, schoolID).Error; err != nil {
		return nil, notFound(err)
	}
	return &file, nil
}

func (s *FileStore) GetByUploader(ctx context.Context, id, schoolID, uploaderID string) (*storage.File, error) {
	var file storage.File
	if err := s.db.WithContext(ctx).First(&file, "id = ? AND school_id = ? AND uploader_id = ?", id, schoolID, uploaderID).Error; err != nil {
		return nil, notFound(err)
	}
	return &file, nil
}

func (s *FileStore) ListByIDs(ctx context.Context, ids []string) ([]storage.File, error) {
	var files []storage.File
	if len(ids) == 0 {
		return files, nil
	}
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}

var _ storage.FileRepository = (*FileStore)(nil)
