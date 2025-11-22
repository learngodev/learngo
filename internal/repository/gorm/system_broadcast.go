package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// SystemBroadcastStore implements SystemBroadcastRepository with GORM.
type SystemBroadcastStore struct {
	db *gorm.DB
}

// NewSystemBroadcastStore builds a SystemBroadcastStore.
func NewSystemBroadcastStore(db *gorm.DB) *SystemBroadcastStore {
	return &SystemBroadcastStore{db: db}
}

func (s *SystemBroadcastStore) EnsureDefaults(ctx context.Context, schoolID string, defaults []domain.SystemBroadcast) error {
	if len(defaults) == 0 {
		return nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&domain.SystemBroadcast{}).Where("school_id = ?", schoolID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	batch := make([]domain.SystemBroadcast, len(defaults))
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

func (s *SystemBroadcastStore) List(ctx context.Context, schoolID string) ([]domain.SystemBroadcast, error) {
	var broadcasts []domain.SystemBroadcast
	err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("sort_order ASC, created_at DESC").
		Find(&broadcasts).Error
	return broadcasts, err
}

func (s *SystemBroadcastStore) Get(ctx context.Context, schoolID, id string) (*domain.SystemBroadcast, error) {
	var b domain.SystemBroadcast
	if err := s.db.WithContext(ctx).Where("school_id = ? AND id = ?", schoolID, id).First(&b).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (s *SystemBroadcastStore) UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*domain.SystemBroadcast, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	updates["updated_at"] = time.Now()

	result := s.db.WithContext(ctx).
		Model(&domain.SystemBroadcast{}).
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

var _ repository.SystemBroadcastRepository = (*SystemBroadcastStore)(nil)
