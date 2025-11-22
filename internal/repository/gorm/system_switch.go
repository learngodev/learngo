package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// SystemSwitchStore implements SystemSwitchRepository using GORM.
type SystemSwitchStore struct {
	db *gorm.DB
}

// NewSystemSwitchStore constructs a SystemSwitchStore instance.
func NewSystemSwitchStore(db *gorm.DB) *SystemSwitchStore {
	return &SystemSwitchStore{db: db}
}

func (s *SystemSwitchStore) EnsureDefaults(ctx context.Context, schoolID string, defaults []domain.SystemSwitch) error {
	if len(defaults) == 0 {
		return nil
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&domain.SystemSwitch{}).Where("school_id = ?", schoolID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	batch := make([]domain.SystemSwitch, len(defaults))
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

func (s *SystemSwitchStore) List(ctx context.Context, schoolID string) ([]domain.SystemSwitch, error) {
	var switches []domain.SystemSwitch
	err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("sort_order ASC, created_at ASC").
		Find(&switches).Error
	return switches, err
}

func (s *SystemSwitchStore) Get(ctx context.Context, schoolID, id string) (*domain.SystemSwitch, error) {
	var sw domain.SystemSwitch
	if err := s.db.WithContext(ctx).Where("school_id = ? AND id = ?", schoolID, id).First(&sw).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &sw, nil
}

func (s *SystemSwitchStore) UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*domain.SystemSwitch, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	updates["updated_at"] = time.Now()

	result := s.db.WithContext(ctx).
		Model(&domain.SystemSwitch{}).
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

var _ repository.SystemSwitchRepository = (*SystemSwitchStore)(nil)
