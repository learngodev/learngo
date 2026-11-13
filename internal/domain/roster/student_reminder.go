package roster

import (
	"time"
)

// StudentReminderPriority 标记提醒的紧急程度。
type StudentReminderPriority string

const (
	StudentReminderPriorityNormal StudentReminderPriority = "normal"
	StudentReminderPriorityHigh   StudentReminderPriority = "high"
)

// StudentReminder 存储学生自建的待办事项。
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
