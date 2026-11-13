package system

import (
	"context"
)

// SystemSwitchRepository 持久化管理员功能开关。
type SystemSwitchRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []SystemSwitch) error
	List(ctx context.Context, schoolID string) ([]SystemSwitch, error)
	Get(ctx context.Context, schoolID, id string) (*SystemSwitch, error)
	UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*SystemSwitch, error)
}

// SystemParameterRepository 持久化可配置参数。
type SystemParameterRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []SystemParameter) error
	List(ctx context.Context, schoolID string) ([]SystemParameter, error)
	Get(ctx context.Context, schoolID, id string) (*SystemParameter, error)
	UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*SystemParameter, error)
}

// SystemBroadcastRepository 持久化公告记录。
type SystemBroadcastRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []SystemBroadcast) error
	List(ctx context.Context, schoolID string) ([]SystemBroadcast, error)
	Get(ctx context.Context, schoolID, id string) (*SystemBroadcast, error)
	UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*SystemBroadcast, error)
}

// SystemAuditRepository 存储系统设置的审计记录。
type SystemAuditRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []SystemAuditLog) error
	Create(ctx context.Context, entry *SystemAuditLog) error
	List(ctx context.Context, schoolID string, limit int) ([]SystemAuditLog, error)
}
