package course

import "context"

// ClassroomRepository manages classroom persistence.
type ClassroomRepository interface {
	Create(ctx context.Context, classroom *Classroom) error
	GetByID(ctx context.Context, id string) (*Classroom, error)
	List(ctx context.Context, schoolID string, page, size int) ([]Classroom, int64, error)
	Update(ctx context.Context, classroom *Classroom) error
	Delete(ctx context.Context, id string) error
}
