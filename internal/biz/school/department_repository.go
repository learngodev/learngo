package school

import "context"

// DepartmentRepository handles departments.
type DepartmentRepository interface {
	Create(ctx context.Context, department *Department) error
	List(ctx context.Context, schoolID string) ([]Department, error)
	GetByID(ctx context.Context, id string) (*Department, error)
	UpdateName(ctx context.Context, id, schoolID, name string) error
	Delete(ctx context.Context, id, schoolID string) error
}
