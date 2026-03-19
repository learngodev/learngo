package course

import "context"

// CourseRepository retrieves course metadata.
type CourseRepository interface {
	Create(ctx context.Context, course *Course) error
	List(ctx context.Context, schoolID string, page, size int) ([]Course, int64, error)
	GetByID(ctx context.Context, id string) (*Course, error)
	Update(ctx context.Context, course *Course) error
	Delete(ctx context.Context, id string) error
	ListByIDs(ctx context.Context, ids []string) ([]Course, error)
	ListAssignments(ctx context.Context, schoolID string, courseID, departmentID, classID string, onlyAssigned bool, page, size int) ([]CourseAssignmentInfo, int64, error)
	ListWithFilters(ctx context.Context, schoolID string, departmentID, classID string, page, size int) ([]Course, int64, error)
	ListAssignmentsByCourseIDs(ctx context.Context, courseIDs []string) ([]CourseAssignmentInfo, error)
	GetByInvitationCode(ctx context.Context, code string) (*Course, error)
	ListByStudentID(ctx context.Context, studentID string) ([]Course, error)
}

// CourseStudentRepository manages student enrollments in courses.
type CourseStudentRepository interface {
	BatchCreate(ctx context.Context, enrollments []CourseStudent) error
	ListByCourseID(ctx context.Context, courseID string) ([]CourseStudent, error)
	DeleteByCourseAndStudent(ctx context.Context, courseID string, studentIDs []string) error
}

// CourseTeacherRepository handles course-teacher associations.
type CourseTeacherRepository interface {
	Add(ctx context.Context, courseID string, teacherIDs []string) error
	Remove(ctx context.Context, courseID string, teacherIDs []string) error
	ListByCourseID(ctx context.Context, courseID string) ([]CourseTeacher, error)
	ListCourseIDsByTeacher(ctx context.Context, teacherID string) ([]string, error)
}
