package system

import "context"

// SystemSwitchRepository persists admin feature toggles.
type SystemSwitchRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []SystemSwitch) error
	List(ctx context.Context, schoolID string) ([]SystemSwitch, error)
	Get(ctx context.Context, schoolID, id string) (*SystemSwitch, error)
	UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*SystemSwitch, error)
}
