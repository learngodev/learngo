package course

import (
	"context"
)

// CourseScheduleRepository manages recurring schedule rules.
type CourseScheduleRepository interface {
	Create(ctx context.Context, schedule *CourseSchedule) error
	Update(ctx context.Context, schedule *CourseSchedule) error
	Delete(ctx context.Context, id string) error
	ListByCourse(ctx context.Context, courseID string) ([]CourseSchedule, error)
	ListBySchool(ctx context.Context, schoolID string) ([]CourseSchedule, error)
	ListDetailsBySchool(ctx context.Context, schoolID string, courseID string) ([]CourseScheduleDetail, error)
	GetStats(ctx context.Context, schoolID string) (*ScheduleStats, error)
	ListByClassroom(ctx context.Context, classroomID string) ([]CourseSchedule, error)
	ListByTeacher(ctx context.Context, teacherID string) ([]CourseSchedule, error)
	ListByClass(ctx context.Context, classID string) ([]CourseSchedule, error)
}
