package identity

import (
	"context"
	"time"
)

// StudentReminderRepository persists custom reminders created by students.
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
