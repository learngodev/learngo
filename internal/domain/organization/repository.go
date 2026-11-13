package organization

import (
	"context"
)

// SchoolRepository 负责学校的持久化。
type SchoolRepository interface {
	List(ctx context.Context) ([]School, error)
}

// ClassroomRepository 负责教室的持久化。
type ClassroomRepository interface {
	Create(ctx context.Context, classroom *Classroom) error
	GetByID(ctx context.Context, id string) (*Classroom, error)
	List(ctx context.Context, schoolID string, page, size int) ([]Classroom, int64, error)
	Update(ctx context.Context, classroom *Classroom) error
	Delete(ctx context.Context, id string) error
}

// DepartmentRepository 负责院系的持久化。
type DepartmentRepository interface {
	Create(ctx context.Context, department *Department) error
	List(ctx context.Context, schoolID string) ([]Department, error)
	GetByID(ctx context.Context, id string) (*Department, error)
	UpdateName(ctx context.Context, id, schoolID, name string) error
	Delete(ctx context.Context, id, schoolID string) error
}

// ClassRepository 负责班级的持久化。
type ClassRepository interface {
	Create(ctx context.Context, class *Class) error
	ListByDepartment(ctx context.Context, schoolID, departmentID string) ([]Class, error)
	GetByID(ctx context.Context, id string) (*Class, error)
	ListByIDs(ctx context.Context, ids []string) ([]Class, error)
	UpdateName(ctx context.Context, id, schoolID, name string) error
	Delete(ctx context.Context, id, schoolID string) error
}
