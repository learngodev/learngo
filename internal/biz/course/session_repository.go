package course

import (
	"context"
	"time"
)

// CourseSessionRepository retrieves scheduled lessons.
type CourseSessionRepository interface {
	Create(ctx context.Context, session *CourseSession) error
	Update(ctx context.Context, session *CourseSession) error
	GetByID(ctx context.Context, id string) (*CourseSession, error)
	ListBetween(ctx context.Context, start, end time.Time) ([]CourseSession, error)
	ListByClassBetween(ctx context.Context, classID string, start, end time.Time) ([]CourseSession, error)
	ListByTeacherBetween(ctx context.Context, teacherID string, start, end time.Time) ([]CourseSession, error)
	Exists(ctx context.Context, courseID, classID string, teacherID *string, slotID string, startsAt time.Time) (bool, error)
}
