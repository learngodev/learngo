package assignment

import (
	"time"

	"learn-go/internal/biz/shared"
)

// AssignmentSubmission stores student answers.
type AssignmentSubmission struct {
	ID           string `gorm:"primaryKey;size:36"`
	AssignmentID string `gorm:"size:36;index"`
	StudentID    string `gorm:"size:36;index"`
	SubmittedAt  *time.Time
	Score        *float64
	Feedback     string `gorm:"type:text"`
	Status       string `gorm:"size:32"`
	Progress     int    `gorm:"default:0"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SubmissionItem holds answer per question.
type SubmissionItem struct {
	ID           string `gorm:"primaryKey;size:36"`
	SubmissionID string `gorm:"size:36;index"`
	QuestionID   string `gorm:"size:36;index"`
	Answer       string `gorm:"type:text"`
	Score        *float64
}

// SubmissionComment allows message on submissions.
type SubmissionComment struct {
	ID            string      `gorm:"primaryKey;size:36"`
	SubmissionID  string      `gorm:"size:36;index"`
	AuthorID      string      `gorm:"size:36;index"`
	AuthorRole    shared.Role `gorm:"size:16"`
	Content       string      `gorm:"type:text"`
	AttachmentURI string      `gorm:"size:256"`
	CreatedAt     time.Time
}

// SubmissionAttachment links files to submissions.
type SubmissionAttachment struct {
	ID           string `gorm:"primaryKey;size:36"`
	SubmissionID string `gorm:"size:36;index"`
	FileID       string `gorm:"size:36;index"`
	CreatedAt    time.Time
}

// AssignmentSubmissionStats holds aggregated submission data per assignment.
type AssignmentSubmissionStats struct {
	AssignmentID      string
	Total             int64
	Submitted         int64
	Graded            int64
	LatestSubmittedAt *time.Time
	AverageScore      *float64
	MaxScore          *float64
	MinScore          *float64
	ScoreDistribution AssignmentScoreDistribution
}

// AssignmentScoreDistribution holds counts for score buckets.
type AssignmentScoreDistribution struct {
	Below60        int64
	Between60And70 int64
	Between70And80 int64
	Between80And90 int64
	Above90        int64
}
