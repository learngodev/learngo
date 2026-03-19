package school

import "context"

// ClassRepository handles classes.
type ClassRepository interface {
	Create(ctx context.Context, class *Class) error
	ListByDepartment(ctx context.Context, schoolID, departmentID string) ([]Class, error)
	GetByID(ctx context.Context, id string) (*Class, error)
	ListByIDs(ctx context.Context, ids []string) ([]Class, error)
	UpdateName(ctx context.Context, id, schoolID, name string) error
	Delete(ctx context.Context, id, schoolID string) error
}
