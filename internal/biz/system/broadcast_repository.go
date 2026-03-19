package system

import "context"

// SystemBroadcastRepository persists announcement records.
type SystemBroadcastRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []SystemBroadcast) error
	List(ctx context.Context, schoolID string) ([]SystemBroadcast, error)
	Get(ctx context.Context, schoolID, id string) (*SystemBroadcast, error)
	UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*SystemBroadcast, error)
}
