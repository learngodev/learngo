package ai

import "context"

// AIAgentSettingRepository manages per-school AI configuration records.
type AIAgentSettingRepository interface {
	GetBySchoolID(ctx context.Context, schoolID string) (*AIAgentSetting, error)
	Upsert(ctx context.Context, setting *AIAgentSetting) error
}

// AIAgentSettingAuditRepository stores administrative audits for AI settings.
type AIAgentSettingAuditRepository interface {
	Create(ctx context.Context, entry *AIAgentSettingAudit) error
	ListRecent(ctx context.Context, schoolID string, limit int) ([]AIAgentSettingAudit, error)
}

// AIUsageLogRepository persists non-chat AI usage logs.
type AIUsageLogRepository interface {
	Create(ctx context.Context, log *AIUsageLog) error
}
