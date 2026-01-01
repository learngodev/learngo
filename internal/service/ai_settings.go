package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

var (
	// ErrAISettingNotConfigured signals that the AI setting is absent.
	ErrAISettingNotConfigured = errors.New("ai setting not configured")
)

// UpdateAIAgentSettingInput collects configuration updates from administrators.
type UpdateAIAgentSettingInput struct {
	SchoolID                string
	Provider                domain.AIProvider
	Model                   string
	APIKey                  string
	BaseURL                 string
	Temperature             float32
	TopP                    float32
	MaxOutputTokens         int
	MaxDailyRequests        int
	MaxConcurrentRequests   int
	MaxConversationMessages int
	SystemPrompt            string
	VisionEnabled           bool
	OperatorID              string
	OperatorName            string
}

// AISettingsService owns AI configuration (settings + audits).
type AISettingsService struct {
	settings repository.AIAgentSettingRepository
	audits   repository.AIAgentSettingAuditRepository
	accounts repository.AccountRepository
}

func NewAISettingsService(
	settings repository.AIAgentSettingRepository,
	audits repository.AIAgentSettingAuditRepository,
	accounts repository.AccountRepository,
) *AISettingsService {
	return &AISettingsService{settings: settings, audits: audits, accounts: accounts}
}

// GetSetting returns the AI setting for a given school (if any).
func (s *AISettingsService) GetSetting(ctx context.Context, schoolID string) (*domain.AIAgentSetting, error) {
	if strings.TrimSpace(schoolID) == "" {
		return nil, errors.New("school_id required")
	}

	setting, err := s.settings.GetBySchoolID(ctx, schoolID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return setting, nil
}

// UpdateSetting creates or updates the AI configuration.
func (s *AISettingsService) UpdateSetting(ctx context.Context, input UpdateAIAgentSettingInput) (*domain.AIAgentSetting, error) {
	// Sanitize API Key
	if input.APIKey != "" {
		input.APIKey = strings.TrimSpace(input.APIKey)
		if strings.HasPrefix(strings.ToLower(input.APIKey), "bearer ") {
			input.APIKey = strings.TrimSpace(input.APIKey[7:])
		}
	}

	if strings.TrimSpace(input.SchoolID) == "" {
		return nil, errors.New("school_id required")
	}
	if strings.TrimSpace(input.OperatorID) == "" {
		return nil, errors.New("operator_id required")
	}
	if input.Provider == "" {
		return nil, errors.New("provider required")
	}

	switch input.Provider {
	case domain.AIProviderQwen, domain.AIProviderDeepSeek:
	default:
		return nil, errors.New("unsupported provider")
	}

	if strings.TrimSpace(input.Model) == "" {
		return nil, errors.New("model required")
	}
	if input.Temperature < 0 || input.Temperature > 2 {
		return nil, errors.New("temperature must be between 0 and 2")
	}
	if input.TopP < 0 || input.TopP > 1 {
		return nil, errors.New("top_p must be between 0 and 1")
	}
	if input.MaxOutputTokens < 0 {
		return nil, errors.New("max_output_tokens must be >= 0")
	}
	if input.MaxDailyRequests < 0 {
		return nil, errors.New("max_daily_requests must be >= 0")
	}
	if input.MaxConcurrentRequests < 0 {
		return nil, errors.New("max_concurrent_requests must be >= 0")
	}
	if input.MaxConversationMessages < 0 {
		return nil, errors.New("max_conversation_messages must be >= 0")
	}

	existing, err := s.settings.GetBySchoolID(ctx, input.SchoolID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	var setting *domain.AIAgentSetting
	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		if input.APIKey == "" {
			return nil, errors.New("api_key required for initial configuration")
		}
		setting = &domain.AIAgentSetting{
			ID:       uuid.NewString(),
			SchoolID: input.SchoolID,
		}
	} else {
		setting = existing
	}

	setting.Provider = input.Provider
	setting.Model = input.Model
	if input.APIKey != "" {
		setting.APIKey = input.APIKey
	}
	setting.BaseURL = input.BaseURL
	setting.Temperature = input.Temperature
	setting.TopP = input.TopP
	setting.MaxOutputTokens = input.MaxOutputTokens
	setting.MaxDailyRequests = input.MaxDailyRequests
	setting.MaxConcurrentRequests = input.MaxConcurrentRequests
	setting.MaxConversationMessages = input.MaxConversationMessages
	setting.SystemPrompt = input.SystemPrompt
	setting.VisionEnabled = input.VisionEnabled
	setting.UpdatedBy = input.OperatorID

	operatorName := strings.TrimSpace(input.OperatorName)
	if operatorName == "" && s.accounts != nil {
		if account, err := s.accounts.FindByID(ctx, input.OperatorID); err == nil && account != nil {
			operatorName = account.DisplayName
		}
	}
	if operatorName == "" {
		operatorName = input.OperatorID
	}
	setting.UpdatedByName = operatorName
	setting.UpdatedAt = now

	if err := s.settings.Upsert(ctx, setting); err != nil {
		return nil, err
	}

	audit := &domain.AIAgentSettingAudit{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		OperatorID:   input.OperatorID,
		OperatorName: operatorName,
		Action:       "update_setting",
		Detail:       fmt.Sprintf("provider=%s, model=%s, vision=%t", setting.Provider, setting.Model, setting.VisionEnabled),
		CreatedAt:    now,
	}
	if err := s.audits.Create(ctx, audit); err != nil {
		return nil, err
	}

	return setting, nil
}

// ListSettingAudits returns recent audit entries for AI configuration.
func (s *AISettingsService) ListSettingAudits(ctx context.Context, schoolID string, limit int) ([]domain.AIAgentSettingAudit, error) {
	if strings.TrimSpace(schoolID) == "" {
		return nil, errors.New("school_id required")
	}
	if limit <= 0 {
		limit = 20
	}
	return s.audits.ListRecent(ctx, schoolID, limit)
}
