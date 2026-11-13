package coursework

import (
	"time"

	"learn-go/internal/domain/identity"
)

// AssignmentSubmission 存储学生的作答。
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

// SubmissionItem 保存单道题的作答。
type SubmissionItem struct {
	ID           string `gorm:"primaryKey;size:36"`
	SubmissionID string `gorm:"size:36;index"`
	QuestionID   string `gorm:"size:36;index"`
	Answer       string `gorm:"type:text"`
	Score        *float64
}

// SubmissionComment 表示针对提交的评论。
type SubmissionComment struct {
	ID            string        `gorm:"primaryKey;size:36"`
	SubmissionID  string        `gorm:"size:36;index"`
	AuthorID      string        `gorm:"size:36;index"`
	AuthorRole    identity.Role `gorm:"size:16"`
	Content       string        `gorm:"type:text"`
	AttachmentURI string        `gorm:"size:256"`
	CreatedAt     time.Time
}

// SubmissionAttachment 关联提交与附件文件。
type SubmissionAttachment struct {
	ID           string `gorm:"primaryKey;size:36"`
	SubmissionID string `gorm:"size:36;index"`
	FileID       string `gorm:"size:36;index"`
	CreatedAt    time.Time
}
