package assignment

import (
	"context"
	"time"

	"learn-go/internal/biz/storage"
)

// AssignmentRepository handles assignments.
type AssignmentRepository interface {
	Create(ctx context.Context, assignment *Assignment, questions []AssignmentQuestion, attachments []AssignmentAttachment) error
	Get(ctx context.Context, id string) (*Assignment, []AssignmentQuestion, []storage.File, error)
	ListByClass(ctx context.Context, classID string, limit int, types []AssignmentType) ([]Assignment, error)
	ListDueBetween(ctx context.Context, classID string, start, end time.Time, types []AssignmentType) ([]Assignment, error)
	ListByTeacher(ctx context.Context, teacherID string, limit int, classID string, types []AssignmentType) ([]Assignment, error)
	ListDueBetweenByTeacher(ctx context.Context, teacherID string, start, end time.Time, types []AssignmentType) ([]Assignment, error)
	Search(ctx context.Context, teacherID string, query string) ([]Assignment, error)
	Update(ctx context.Context, assignment *Assignment) error
}
