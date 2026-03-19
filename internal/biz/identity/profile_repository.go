package identity

import "context"

// TeacherRepository handles teacher profile persistence.
type TeacherRepository interface {
	Create(ctx context.Context, teacher *Teacher) error
	GetByNumber(ctx context.Context, schoolID, number string) (*Teacher, error)
	GetByID(ctx context.Context, id string) (*Teacher, error)
	GetByAccountID(ctx context.Context, accountID string) (*Teacher, error)
	Update(ctx context.Context, teacher *Teacher) error
}

// StudentRepository handles student profile persistence.
type StudentRepository interface {
	Create(ctx context.Context, student *Student) error
	GetByNumber(ctx context.Context, schoolID, number string) (*Student, error)
	GetByID(ctx context.Context, id string) (*Student, error)
	GetByAccountID(ctx context.Context, accountID string) (*Student, error)
	ListByIDs(ctx context.Context, ids []string) ([]Student, error)
	ListByClassID(ctx context.Context, classID string) ([]Student, error)
	ListByDepartmentID(ctx context.Context, departmentID string) ([]Student, error)
	CountByClassIDs(ctx context.Context, classIDs []string) (map[string]int64, error)
	UpdateClassID(ctx context.Context, studentID string, classID string) error
	Update(ctx context.Context, student *Student) error
}

// TeacherStudentRepository manages relationships between teachers and students.
type TeacherStudentRepository interface {
	BindTeachers(ctx context.Context, studentID string, teacherIDs []string) error
}
