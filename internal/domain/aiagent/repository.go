package aiagent

import (
	"context"
)

// AIUsageLogRepository 持久化非对话类 AI 的使用日志。
type AIUsageLogRepository interface {
	Create(ctx context.Context, log *AIUsageLog) error
}

// AIAgentSettingRepository 管理各学校的 AI 配置记录。
type AIAgentSettingRepository interface {
	GetBySchoolID(ctx context.Context, schoolID string) (*AIAgentSetting, error)
	Upsert(ctx context.Context, setting *AIAgentSetting) error
}

// AIAgentSettingAuditRepository 存储 AI 配置的管理审计记录。
type AIAgentSettingAuditRepository interface {
	Create(ctx context.Context, entry *AIAgentSettingAudit) error
	ListRecent(ctx context.Context, schoolID string, limit int) ([]AIAgentSettingAudit, error)
}
