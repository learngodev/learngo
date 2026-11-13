package gormrepo

import (
	"context"
	"errors"

	"learn-go/internal/domain/aiagent"

	"gorm.io/gorm"
)

type AIUsageLogStore struct {
	db *gorm.DB
}

func NewAIUsageLogStore(db *gorm.DB) *AIUsageLogStore {
	return &AIUsageLogStore{db: db}
}

func (s *AIUsageLogStore) Create(ctx context.Context, log *aiagent.AIUsageLog) error {
	if log == nil {
		return errors.New("log required")
	}
	if log.ID == "" {
		return errors.New("log id required")
	}
	return s.db.WithContext(ctx).Create(log).Error
}

var _ aiagent.AIUsageLogRepository = (*AIUsageLogStore)(nil)
