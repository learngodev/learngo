package gormrepo

import (
	"context"
	"errors"
	"gorm.io/gorm"
	aibiz "learn-go/internal/biz/ai"
)

type AIUsageLogStore struct {
	db *gorm.DB
}

func NewAIUsageLogStore(db *gorm.DB) *AIUsageLogStore {
	return &AIUsageLogStore{db: db}
}

func (s *AIUsageLogStore) Create(ctx context.Context, log *aibiz.AIUsageLog) error {
	if log == nil {
		return errors.New("log required")
	}
	if log.ID == "" {
		return errors.New("log id required")
	}
	return s.db.WithContext(ctx).Create(log).Error
}

var _ aibiz.AIUsageLogRepository = (*AIUsageLogStore)(nil)
