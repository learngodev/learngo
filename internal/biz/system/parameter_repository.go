package system

import "context"

// SystemParameterRepository persists configurable parameters.
type SystemParameterRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []SystemParameter) error
	List(ctx context.Context, schoolID string) ([]SystemParameter, error)
	Get(ctx context.Context, schoolID, id string) (*SystemParameter, error)
	UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*SystemParameter, error)
}
