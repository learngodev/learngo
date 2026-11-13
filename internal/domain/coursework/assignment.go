package coursework

import (
	"time"
)

// AssignmentType 枚举作业/考试类型。
type AssignmentType string

const (
	AssignmentHomework AssignmentType = "homework"
	AssignmentExam     AssignmentType = "exam"
)

// Assignment 表示一次作业或考试。
type Assignment struct {
	ID            string         `gorm:"primaryKey;size:36"`
	CourseID      string         `gorm:"size:36;index"`
	TeacherID     string         `gorm:"size:36;index"`
	ClassID       string         `gorm:"size:36;index"`
	Type          AssignmentType `gorm:"size:16;index"`
	Title         string         `gorm:"size:256"`
	Description   string         `gorm:"size:1024"`
	StartAt       *time.Time
	DueAt         *time.Time
	MaxScore      float64
	AllowResubmit bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// AssignmentAttachment 关联作业与附件文件。
type AssignmentAttachment struct {
	ID           string `gorm:"primaryKey;size:36"`
	AssignmentID string `gorm:"size:36;index"`
	FileID       string `gorm:"size:36;index"`
	CreatedAt    time.Time
}
