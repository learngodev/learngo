package roster

import (
	"context"
	"time"
)

// TeacherRepository 负责教师档案的持久化。
type TeacherRepository interface {
	Create(ctx context.Context, teacher *Teacher) error
	GetByNumber(ctx context.Context, schoolID, number string) (*Teacher, error)
	GetByID(ctx context.Context, id string) (*Teacher, error)
	GetByAccountID(ctx context.Context, accountID string) (*Teacher, error)
	Update(ctx context.Context, teacher *Teacher) error
}

// StudentRepository 负责学生档案的持久化。
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

// StudentReminderRepository 持久化学生自建的提醒事项。
type StudentReminderRepository interface {
	ListByStudent(ctx context.Context, studentID string) ([]StudentReminder, error)
	GetByID(ctx context.Context, id string) (*StudentReminder, error)
	Create(ctx context.Context, reminder *StudentReminder) error
	UpdateFields(ctx context.Context, id string, studentID string, updates map[string]any) (*StudentReminder, error)
	Delete(ctx context.Context, id string, studentID string) error
	MarkAllCompleted(ctx context.Context, studentID string, completed bool, timestamp *time.Time) error
	SetCompletion(ctx context.Context, id string, studentID string, completed bool, timestamp *time.Time) (*StudentReminder, error)
	MarkBatchCompleted(ctx context.Context, studentID string, ids []string, completed bool, timestamp *time.Time) error
}

// TeacherStudentRepository 管理教师与学生之间的关系。
type TeacherStudentRepository interface {
	BindTeachers(ctx context.Context, studentID string, teacherIDs []string) error
}
