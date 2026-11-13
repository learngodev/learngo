package usecase

import (
	"context"
	"errors"
	"fmt"
	"learn-go/internal/domain"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain/aiagent"
	"learn-go/internal/domain/identity"
)

var (
	// ErrAISettingNotConfigured 表示尚未配置 AI。
	ErrAISettingNotConfigured = errors.New("ai setting not configured")
)

// UpdateAIAgentSettingInput 收集管理员提交的配置更新。
type UpdateAIAgentSettingInput struct {
	SchoolID                string
	Provider                aiagent.AIProvider
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

// AISettingsUsecase 负责 AI 配置（配置项与审计记录）。
type AISettingsUsecase struct {
	settings aiagent.AIAgentSettingRepository
	audits   aiagent.AIAgentSettingAuditRepository
	accounts identity.AccountRepository
}

func NewAISettingsUsecase(
	settings aiagent.AIAgentSettingRepository,
	audits aiagent.AIAgentSettingAuditRepository,
	accounts identity.AccountRepository,
) *AISettingsUsecase {
	return &AISettingsUsecase{settings: settings, audits: audits, accounts: accounts}
}

// GetSetting 返回指定学校的 AI 配置（如有）。
func (s *AISettingsUsecase) GetSetting(ctx context.Context, schoolID string) (*aiagent.AIAgentSetting, error) {
	if strings.TrimSpace(schoolID) == "" {
		return nil, errors.New("school_id required")
	}

	setting, err := s.settings.GetBySchoolID(ctx, schoolID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return setting, nil
}

// UpdateSetting 创建或更新 AI 配置。
func (s *AISettingsUsecase) UpdateSetting(ctx context.Context, input UpdateAIAgentSettingInput) (*aiagent.AIAgentSetting, error) {
	// 清理 API Key
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
	case aiagent.AIProviderQwen, aiagent.AIProviderDeepSeek:
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
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	now := time.Now()
	var setting *aiagent.AIAgentSetting
	if existing == nil || errors.Is(err, domain.ErrNotFound) {
		if input.APIKey == "" {
			return nil, errors.New("api_key required for initial configuration")
		}
		setting = &aiagent.AIAgentSetting{
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

	audit := &aiagent.AIAgentSettingAudit{
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

// ListSettingAudits 返回 AI 配置最近的审计记录。
func (s *AISettingsUsecase) ListSettingAudits(ctx context.Context, schoolID string, limit int) ([]aiagent.AIAgentSettingAudit, error) {
	if strings.TrimSpace(schoolID) == "" {
		return nil, errors.New("school_id required")
	}
	if limit <= 0 {
		limit = 20
	}
	return s.audits.ListRecent(ctx, schoolID, limit)
}
