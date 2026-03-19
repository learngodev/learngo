package assignment

import "time"

// AssignmentType enumerates assignment and exam types.
type AssignmentType string

const (
	AssignmentHomework AssignmentType = "homework"
	AssignmentExam     AssignmentType = "exam"
)

// QuestionType enumerates supported question formats.
type QuestionType string

const (
	QuestionFill      QuestionType = "fill"
	QuestionChoice    QuestionType = "choice"
	QuestionJudgement QuestionType = "judge"
	QuestionEssay     QuestionType = "essay"
)

// Assignment represents homework or exam.
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

// AssignmentQuestion holds task content.
type AssignmentQuestion struct {
	ID           string       `gorm:"primaryKey;size:36"`
	AssignmentID string       `gorm:"size:36;index"`
	Type         QuestionType `gorm:"size:16"`
	Prompt       string       `gorm:"type:text"`
	Options      string       `gorm:"type:text"`
	Answer       string       `gorm:"type:text"`
	Score        float64
	OrderIndex   int
}

// AssignmentAttachment links files to assignments.
type AssignmentAttachment struct {
	ID           string `gorm:"primaryKey;size:36"`
	AssignmentID string `gorm:"size:36;index"`
	FileID       string `gorm:"size:36;index"`
	CreatedAt    time.Time
}
