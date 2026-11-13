package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/domain/system"

	"gorm.io/gorm"
)

// SystemParameterStore 基于 GORM 实现 system.SystemParameterRepository。
type SystemParameterStore struct {
	db *gorm.DB
}

// NewSystemParameterStore 构造 SystemParameterStore。
func NewSystemParameterStore(db *gorm.DB) *SystemParameterStore {
	return &SystemParameterStore{db: db}
}

func (s *SystemParameterStore) EnsureDefaults(ctx context.Context, schoolID string, defaults []system.SystemParameter) error {
	if len(defaults) == 0 {
		return nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&system.SystemParameter{}).Where("school_id = ?", schoolID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	batch := make([]system.SystemParameter, len(defaults))
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

func (s *SystemParameterStore) List(ctx context.Context, schoolID string) ([]system.SystemParameter, error) {
	var params []system.SystemParameter
	err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("sort_order ASC, created_at ASC").
		Find(&params).Error
	return params, err
}

func (s *SystemParameterStore) Get(ctx context.Context, schoolID, id string) (*system.SystemParameter, error) {
	var param system.SystemParameter
	if err := s.db.WithContext(ctx).Where("school_id = ? AND id = ?", schoolID, id).First(&param).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &param, nil
}

func (s *SystemParameterStore) UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*system.SystemParameter, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	updates["updated_at"] = time.Now()

	result := s.db.WithContext(ctx).
		Model(&system.SystemParameter{}).
		Where("school_id = ? AND id = ?", schoolID, id).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	return s.Get(ctx, schoolID, id)
}

var _ system.SystemParameterRepository = (*SystemParameterStore)(nil)
