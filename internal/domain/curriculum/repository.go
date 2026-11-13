package curriculum

import (
	"context"

	"learn-go/internal/domain/storage"
)

// CourseRepository 负责课程信息的持久化与查询。
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

// CourseStudentRepository 管理学生选课关系。
type CourseStudentRepository interface {
	BatchCreate(ctx context.Context, enrollments []CourseStudent) error
	ListByCourseID(ctx context.Context, courseID string) ([]CourseStudent, error)
	DeleteByCourseAndStudent(ctx context.Context, courseID string, studentIDs []string) error
}

// CourseTeacherRepository 管理课程与教师的关联。
type CourseTeacherRepository interface {
	Add(ctx context.Context, courseID string, teacherIDs []string) error
	Remove(ctx context.Context, courseID string, teacherIDs []string) error
	ListByCourseID(ctx context.Context, courseID string) ([]CourseTeacher, error)
	ListCourseIDsByTeacher(ctx context.Context, teacherID string) ([]string, error)
}

// CourseChapterRepository 管理课程的课后学习章节。
type CourseChapterRepository interface {
	ListByCourse(ctx context.Context, courseID string) ([]CourseChapter, error)
	GetByID(ctx context.Context, courseID string, chapterID string) (*CourseChapter, error)
	ListFiles(ctx context.Context, chapterID string) ([]storage.File, error)
	Create(ctx context.Context, chapter *CourseChapter) error
	UpdateFields(ctx context.Context, courseID string, chapterID string, updates map[string]any) (*CourseChapter, error)
	Delete(ctx context.Context, courseID string, chapterID string) error
	AttachFile(ctx context.Context, chapterID string, fileID string) error
	DetachFile(ctx context.Context, chapterID string, fileID string) error
}
