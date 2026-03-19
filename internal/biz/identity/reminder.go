package identity

import "time"

// StudentReminderPriority marks reminder urgency.
type StudentReminderPriority string

const (
	StudentReminderPriorityNormal StudentReminderPriority = "normal"
	StudentReminderPriorityHigh   StudentReminderPriority = "high"
)

// StudentReminder stores custom tasks created by students.
type StudentReminder struct {
	ID          string `gorm:"primaryKey;size:36"`
	StudentID   string `gorm:"size:36;index"`
	Title       string `gorm:"size:128"`
	Description string `gorm:"size:256"`
	TimeLabel   string `gorm:"size:64"`
	DueAt       *time.Time
	Route       string                  `gorm:"size:128"`
	Priority    StudentReminderPriority `gorm:"size:16;default:'normal'"`
	Icon        string                  `gorm:"size:64"`
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
