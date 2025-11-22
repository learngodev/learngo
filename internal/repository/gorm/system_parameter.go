package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// SystemParameterStore implements SystemParameterRepository with GORM.
type SystemParameterStore struct {
	db *gorm.DB
}

// NewSystemParameterStore builds a SystemParameterStore.
func NewSystemParameterStore(db *gorm.DB) *SystemParameterStore {
	return &SystemParameterStore{db: db}
}

func (s *SystemParameterStore) EnsureDefaults(ctx context.Context, schoolID string, defaults []domain.SystemParameter) error {
	if len(defaults) == 0 {
		return nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&domain.SystemParameter{}).Where("school_id = ?", schoolID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	batch := make([]domain.SystemParameter, len(defaults))
	now := time.Now()
	for i := range defaults {
		batch[i] = defaults[i]
		batch[i].SchoolID = schoolID
		if batch[i].CreatedAt.IsZero() {
			batch[i].CreatedAt = now
		}
		if batch[i].UpdatedAt.IsZero() {
			batch[i].UpdatedAt = now
		}
	}
	return s.db.WithContext(ctx).Create(&batch).Error
}

func (s *SystemParameterStore) List(ctx context.Context, schoolID string) ([]domain.SystemParameter, error) {
	var params []domain.SystemParameter
	err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("sort_order ASC, created_at ASC").
		Find(&params).Error
	return params, err
}

func (s *SystemParameterStore) Get(ctx context.Context, schoolID, id string) (*domain.SystemParameter, error) {
	var param domain.SystemParameter
	if err := s.db.WithContext(ctx).Where("school_id = ? AND id = ?", schoolID, id).First(&param).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &param, nil
}

func (s *SystemParameterStore) UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*domain.SystemParameter, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	updates["updated_at"] = time.Now()

	result := s.db.WithContext(ctx).
		Model(&domain.SystemParameter{}).
		Where("school_id = ? AND id = ?", schoolID, id).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	return s.Get(ctx, schoolID, id)
}

var _ repository.SystemParameterRepository = (*SystemParameterStore)(nil)
