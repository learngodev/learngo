package system

import "context"

// SystemAuditRepository stores audit trail for system settings.
type SystemAuditRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []SystemAuditLog) error
	Create(ctx context.Context, entry *SystemAuditLog) error
	List(ctx context.Context, schoolID string, limit int) ([]SystemAuditLog, error)
}
