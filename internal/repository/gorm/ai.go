package gormrepo

import (
	"context"
	"errors"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// AIAgentSettingStore implements repository.AIAgentSettingRepository.
type AIAgentSettingStore struct {
	db *gorm.DB
}

// NewAIAgentSettingStore constructs an AIAgentSettingStore.
func NewAIAgentSettingStore(db *gorm.DB) *AIAgentSettingStore {
	return &AIAgentSettingStore{db: db}
}

func (s *AIAgentSettingStore) GetBySchoolID(ctx context.Context, schoolID string) (*domain.AIAgentSetting, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	var setting domain.AIAgentSetting
	if err := s.db.WithContext(ctx).Where("school_id = ?", schoolID).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (s *AIAgentSettingStore) Upsert(ctx context.Context, setting *domain.AIAgentSetting) error {
	if setting == nil {
		return errors.New("setting required")
	}
	if setting.ID == "" {
		return errors.New("setting id required")
	}

	var existing domain.AIAgentSetting
	err := s.db.WithContext(ctx).First(&existing, "id = ?", setting.ID).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return s.db.WithContext(ctx).Create(setting).Error
	case err != nil:
		return err
	default:
		updates := map[string]any{
			"provider":                  setting.Provider,
			"model":                     setting.Model,
			"api_key":                   setting.APIKey,
			"base_url":                  setting.BaseURL,
			"temperature":               setting.Temperature,
			"top_p":                     setting.TopP,
			"max_output_tokens":         setting.MaxOutputTokens,
			"max_daily_requests":        setting.MaxDailyRequests,
			"max_concurrent_requests":   setting.MaxConcurrentRequests,
			"max_conversation_messages": setting.MaxConversationMessages,
			"system_prompt":             setting.SystemPrompt,
			"vision_enabled":            setting.VisionEnabled,
			"updated_by":                setting.UpdatedBy,
			"updated_by_name":           setting.UpdatedByName,
			"updated_at":                setting.UpdatedAt,
		}

		return s.db.WithContext(ctx).
			Model(&domain.AIAgentSetting{}).
			Where("id = ?", setting.ID).
			Updates(updates).Error
	}
}

var _ repository.AIAgentSettingRepository = (*AIAgentSettingStore)(nil)

// AIAgentSettingAuditStore implements repository.AIAgentSettingAuditRepository.
type AIAgentSettingAuditStore struct {
	db *gorm.DB
}

// NewAIAgentSettingAuditStore constructs an audit store.
func NewAIAgentSettingAuditStore(db *gorm.DB) *AIAgentSettingAuditStore {
	return &AIAgentSettingAuditStore{db: db}
}

func (s *AIAgentSettingAuditStore) Create(ctx context.Context, entry *domain.AIAgentSettingAudit) error {
	if entry == nil {
		return errors.New("entry required")
	}
	if entry.ID == "" {
		return errors.New("entry id required")
	}
	return s.db.WithContext(ctx).Create(entry).Error
}

func (s *AIAgentSettingAuditStore) ListRecent(ctx context.Context, schoolID string, limit int) ([]domain.AIAgentSettingAudit, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	if limit <= 0 {
		limit = 20
	}

	var entries []domain.AIAgentSettingAudit
	query := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("created_at DESC").
		Limit(limit)
	if err := query.Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

var _ repository.AIAgentSettingAuditRepository = (*AIAgentSettingAuditStore)(nil)
