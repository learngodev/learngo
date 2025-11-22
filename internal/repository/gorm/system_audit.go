package gormrepo

import (
	"context"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// SystemAuditStore implements SystemAuditRepository with GORM.
type SystemAuditStore struct {
	db *gorm.DB
}

// NewSystemAuditStore constructs a SystemAuditStore.
func NewSystemAuditStore(db *gorm.DB) *SystemAuditStore {
	return &SystemAuditStore{db: db}
}

func (s *SystemAuditStore) EnsureDefaults(ctx context.Context, schoolID string, defaults []domain.SystemAuditLog) error {
	if len(defaults) == 0 {
		return nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&domain.SystemAuditLog{}).Where("school_id = ?", schoolID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	batch := make([]domain.SystemAuditLog, len(defaults))
	now := time.Now()
	for i := range defaults {
		batch[i] = defaults[i]
		batch[i].SchoolID = schoolID
		if batch[i].ID == "" {
			batch[i].ID = uuid.NewString()
		}
		if batch[i].CreatedAt.IsZero() {
			batch[i].CreatedAt = now
		}
		if batch[i].TimeLabel == "" {
			batch[i].TimeLabel = batch[i].CreatedAt.Format("01-02 15:04")
		}
	}
	return s.db.WithContext(ctx).Create(&batch).Error
}

func (s *SystemAuditStore) Create(ctx context.Context, entry *domain.SystemAuditLog) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	if entry.TimeLabel == "" {
		entry.TimeLabel = entry.CreatedAt.Format("01-02 15:04")
	}
	return s.db.WithContext(ctx).Create(entry).Error
}

func (s *SystemAuditStore) List(ctx context.Context, schoolID string, limit int) ([]domain.SystemAuditLog, error) {
	query := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var logs []domain.SystemAuditLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

var _ repository.SystemAuditRepository = (*SystemAuditStore)(nil)
