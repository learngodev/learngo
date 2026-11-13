package scheduling

import (
	"context"
	"time"
)

// CourseScheduleRepository 管理周期性排课规则。
type CourseScheduleRepository interface {
	Create(ctx context.Context, schedule *CourseSchedule) error
	Update(ctx context.Context, schedule *CourseSchedule) error
	GetByID(ctx context.Context, id string) (*CourseSchedule, error)
	Delete(ctx context.Context, id string) error
	ListByCourse(ctx context.Context, courseID string) ([]CourseSchedule, error)
	ListBySchool(ctx context.Context, schoolID string) ([]CourseSchedule, error)
	ListDetailsBySchool(ctx context.Context, schoolID string, courseID string) ([]CourseScheduleDetail, error)
	GetStats(ctx context.Context, schoolID string) (*ScheduleStats, error)
	ListByClassroom(ctx context.Context, classroomID string) ([]CourseSchedule, error)
	ListByTeacher(ctx context.Context, teacherID string) ([]CourseSchedule, error)
	ListByClass(ctx context.Context, classID string) ([]CourseSchedule, error)
}

// CourseSessionRepository 负责已排课次的持久化与查询。
type CourseSessionRepository interface {
	Create(ctx context.Context, session *CourseSession) error
	Update(ctx context.Context, session *CourseSession) error
	DeleteByIDs(ctx context.Context, ids []string) error
	GetByID(ctx context.Context, id string) (*CourseSession, error)
	ListBetween(ctx context.Context, start, end time.Time) ([]CourseSession, error)
	ListByClassBetween(ctx context.Context, classID string, start, end time.Time) ([]CourseSession, error)
	ListByTeacherBetween(ctx context.Context, teacherID string, start, end time.Time) ([]CourseSession, error)
	Exists(ctx context.Context, courseID, classID string, teacherID *string, slotID string, startsAt time.Time) (bool, error)
}

// TimeSlotRepository 管理上课节次。
type TimeSlotRepository interface {
	Create(ctx context.Context, slot *TimeSlot) error
	Update(ctx context.Context, slot *TimeSlot) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, schoolID string) ([]TimeSlot, error)
	FindByID(ctx context.Context, id string) (*TimeSlot, error)
}

// CourseSlotRepository 管理课程节次（目前是 TimeSlotRepository 的别名）。
type CourseSlotRepository interface {
	ListByIDs(ctx context.Context, ids []string) ([]CourseSlot, error)
}
