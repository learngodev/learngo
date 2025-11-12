package domain

import (
	"time"

	"gorm.io/gorm"
)

// Role defines platform roles.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

// AccountStatus indicates account availability state.
type AccountStatus string

const (
	AccountStatusActive                AccountStatus = "active"
	AccountStatusLocked                AccountStatus = "locked"
	AccountStatusPasswordResetRequired AccountStatus = "password_reset_required"
)

// Account represents login credentials tied to a user type.
type Account struct {
	ID           string        `gorm:"primaryKey;size:36"`
	SchoolID     string        `gorm:"size:36;index"`
	Role         Role          `gorm:"size:16;index"`
	Status       AccountStatus `gorm:"size:32;index;default:'active'"`
	Identifier   string        `gorm:"size:64;uniqueIndex"` // teacher number or student number
	PasswordHash string        `gorm:"size:128"`
	DisplayName  string        `gorm:"size:128"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// School represents a tenant scope.
type School struct {
	ID        string `gorm:"primaryKey;size:36"`
	Name      string `gorm:"size:128;unique"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Department groups classes by subject or grade.
type Department struct {
	ID        string `gorm:"primaryKey;size:36"`
	SchoolID  string `gorm:"size:36;index"`
	Name      string `gorm:"size:128"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Class is a teaching class under a department.
type Class struct {
	ID           string  `gorm:"primaryKey;size:36"`
	SchoolID     string  `gorm:"size:36;index"`
	DepartmentID string  `gorm:"size:36;index"`
	Name         string  `gorm:"size:128"`
	HomeroomID   *string `gorm:"size:36"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Teacher profile.
type Teacher struct {
	ID        string `gorm:"primaryKey;size:36"`
	SchoolID  string `gorm:"size:36;index"`
	AccountID string `gorm:"size:36;uniqueIndex"`
	Number    string `gorm:"size:64;uniqueIndex"`
	Email     string `gorm:"size:128"`
	Phone     string `gorm:"size:32"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Student profile.
type Student struct {
	ID        string `gorm:"primaryKey;size:36"`
	SchoolID  string `gorm:"size:36;index"`
	AccountID string `gorm:"size:36;uniqueIndex"`
	Number    string `gorm:"size:64;uniqueIndex"`
	ClassID   string `gorm:"size:36;index"`
	Email     string `gorm:"size:128"`
	Phone     string `gorm:"size:32"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TeacherStudentLink binds teacher(s) to student.
type TeacherStudentLink struct {
	ID        string `gorm:"primaryKey;size:36"`
	TeacherID string `gorm:"size:36;index"`
	StudentID string `gorm:"size:36;index"`
	CreatedAt time.Time
}

// CourseSlot defines a lesson time window.
type CourseSlot struct {
	ID        string `gorm:"primaryKey;size:36"`
	SchoolID  string `gorm:"size:36;index"`
	Name      string `gorm:"size:64"`
	StartTime time.Time
	EndTime   time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Course represents a subject taught by a teacher.
type Course struct {
	ID          string `gorm:"primaryKey;size:36"`
	SchoolID    string `gorm:"size:36;index"`
	Name        string `gorm:"size:128"`
	Description string `gorm:"size:512"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CourseSession is a scheduled lesson.
type CourseSession struct {
	ID        string `gorm:"primaryKey;size:36"`
	CourseID  string `gorm:"size:36;index"`
	ClassID   string `gorm:"size:36;index"`
	TeacherID string `gorm:"size:36;index"`
	SlotID    string `gorm:"size:36;index"`
	StartsAt  time.Time
	EndsAt    time.Time
	Source    string `gorm:"size:32"` // system or teacher
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AssignmentType enumerates assignment/exam types.
type AssignmentType string

const (
	AssignmentHomework AssignmentType = "homework"
	AssignmentExam     AssignmentType = "exam"
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

// QuestionType enumerates supported question formats.
type QuestionType string

const (
	QuestionFill      QuestionType = "fill"
	QuestionChoice    QuestionType = "choice"
	QuestionJudgement QuestionType = "judge"
	QuestionEssay     QuestionType = "essay"
)

// AssignmentQuestion holds task content.
type AssignmentQuestion struct {
	ID           string       `gorm:"primaryKey;size:36"`
	AssignmentID string       `gorm:"size:36;index"`
	Type         QuestionType `gorm:"size:16"`
	Prompt       string       `gorm:"type:text"`
	Options      string       `gorm:"type:text"` // JSON encoded for choice
	Answer       string       `gorm:"type:text"`
	Score        float64
	OrderIndex   int
}

// AssignmentSubmission stores student answers.
type AssignmentSubmission struct {
	ID           string `gorm:"primaryKey;size:36"`
	AssignmentID string `gorm:"size:36;index"`
	StudentID    string `gorm:"size:36;index"`
	SubmittedAt  *time.Time
	Score        *float64
	Feedback     string `gorm:"type:text"`
	Status       string `gorm:"size:32"`
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
	ID            string `gorm:"primaryKey;size:36"`
	SubmissionID  string `gorm:"size:36;index"`
	AuthorID      string `gorm:"size:36;index"`
	AuthorRole    Role   `gorm:"size:16"`
	Content       string `gorm:"type:text"`
	AttachmentURI string `gorm:"size:256"`
	CreatedAt     time.Time
}

// Conversation represents chat channel.
type Conversation struct {
	ID        string `gorm:"primaryKey;size:36"`
	Type      string `gorm:"size:16"` // direct or group
	SchoolID  string `gorm:"size:36;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConversationMember ties users to conversations.
type ConversationMember struct {
	ID             string `gorm:"primaryKey;size:36"`
	ConversationID string `gorm:"size:36;index"`
	AccountID      string `gorm:"size:36;index"`
	Role           Role   `gorm:"size:16"`
	CreatedAt      time.Time
}

// Message holds chat content.
type Message struct {
	ID             string `gorm:"primaryKey;size:36"`
	ConversationID string `gorm:"size:36;index"`
	SenderID       string `gorm:"size:36;index"`
	SenderRole     Role   `gorm:"size:16"`
	Kind           string `gorm:"size:16"` // text,image,video,audio
	Text           string `gorm:"type:text"`
	MediaURI       string `gorm:"size:256"`
	Metadata       string `gorm:"type:text"`
	CreatedAt      time.Time
}

// MessageReceipt tracks read state.
type MessageReceipt struct {
	ID        string `gorm:"primaryKey;size:36"`
	MessageID string `gorm:"size:36;index"`
	AccountID string `gorm:"size:36;index"`
	ReadAt    *time.Time
	CreatedAt time.Time
}

// OssCredential stores object storage credential metadata for administrators.
type OssCredential struct {
	ID                   string `gorm:"primaryKey;size:36"`
	SchoolID             string `gorm:"size:36;index"`
	Name                 string `gorm:"size:128"`
	Endpoint             string `gorm:"size:128"`
	Region               string `gorm:"size:64"`
	Bucket               string `gorm:"size:128"`
	AccessKeyDisplay     string `gorm:"size:128"`
	DirectoryPrefix      string `gorm:"size:128"`
	AllowPublicRead      bool   `gorm:"default:false"`
	AllowMultipartUpload bool   `gorm:"default:false"`
	IsPrimary            bool   `gorm:"index"`
	Active               bool   `gorm:"index"`
	LastRotatedAt        *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// OssPolicyStatus indicates policy enforcement state.
type OssPolicyStatus string

const (
	OssPolicyStatusEnabled  OssPolicyStatus = "enabled"
	OssPolicyStatusReadOnly OssPolicyStatus = "read_only"
	OssPolicyStatusDisabled OssPolicyStatus = "disabled"
)

// OssPolicy describes logical access restrictions for different scenarios.
type OssPolicy struct {
	ID            string          `gorm:"primaryKey;size:36"`
	SchoolID      string          `gorm:"size:36;index"`
	Name          string          `gorm:"size:128"`
	Description   string          `gorm:"size:512"`
	AppliesTo     string          `gorm:"size:128"`
	Status        OssPolicyStatus `gorm:"size:32;index"`
	LastUpdatedAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OssAuditLog records administrative changes made to OSS settings.
type OssAuditLog struct {
	ID           string    `gorm:"primaryKey;size:36"`
	SchoolID     string    `gorm:"size:36;index"`
	Action       string    `gorm:"size:128"`
	OperatorID   string    `gorm:"size:36"`
	OperatorName string    `gorm:"size:128"`
	Detail       string    `gorm:"size:512"`
	CreatedAt    time.Time `gorm:"index"`
}

// Note stores personal or shared notes.
type Note struct {
	ID         string `gorm:"primaryKey;size:36"`
	SchoolID   string `gorm:"size:36;index"`
	OwnerID    string `gorm:"size:36;index"`
	OwnerRole  Role   `gorm:"size:16"`
	Title      string `gorm:"size:256"`
	Content    string `gorm:"type:text"`
	Visibility string `gorm:"size:16"` // private, class, school
	Status     string `gorm:"size:16"` // draft, published
	DeletedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NoteComment for collaborative feedback.
type NoteComment struct {
	ID         string `gorm:"primaryKey;size:36"`
	NoteID     string `gorm:"size:36;index"`
	AuthorID   string `gorm:"size:36;index"`
	AuthorRole Role   `gorm:"size:16"`
	Content    string `gorm:"type:text"`
	CreatedAt  time.Time
}

// AIProvider represents supported AI model providers.
type AIProvider string

const (
	AIProviderQwen     AIProvider = "qwen"
	AIProviderDeepSeek AIProvider = "deepseek"
)

// AIAgentSetting stores per-school AI assistant configuration.
type AIAgentSetting struct {
	ID                      string     `gorm:"primaryKey;size:36"`
	SchoolID                string     `gorm:"size:36;index"`
	Provider                AIProvider `gorm:"size:32"`
	Model                   string     `gorm:"size:128"`
	APIKey                  string     `gorm:"size:256"`
	BaseURL                 string     `gorm:"size:256"`
	Temperature             float32    `gorm:"type:real"`
	TopP                    float32    `gorm:"type:real"`
	MaxOutputTokens         int        `gorm:"default:0"`
	MaxDailyRequests        int        `gorm:"default:0"`
	MaxConcurrentRequests   int        `gorm:"default:0"`
	MaxConversationMessages int        `gorm:"default:0"`
	SystemPrompt            string     `gorm:"type:text"`
	VisionEnabled           bool       `gorm:"default:false"`
	UpdatedBy               string     `gorm:"size:36"`
	UpdatedByName           string     `gorm:"size:128"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// AIAgentSettingAudit tracks administrative changes to AI assistant settings.
type AIAgentSettingAudit struct {
	ID           string    `gorm:"primaryKey;size:36"`
	SchoolID     string    `gorm:"size:36;index"`
	OperatorID   string    `gorm:"size:36"`
	OperatorName string    `gorm:"size:128"`
	Action       string    `gorm:"size:64"`
	Detail       string    `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"index"`
}

// AIChatSession groups student AI assistant interactions.
type AIChatSession struct {
	ID            string    `gorm:"primaryKey;size:36"`
	SchoolID      string    `gorm:"size:36;index"`
	AccountID     string    `gorm:"size:36;index"`
	Role          Role      `gorm:"size:16"`
	Title         string    `gorm:"size:256"`
	LastMessageAt time.Time `gorm:"index"`
	MessageCount  int       `gorm:"default:0"`
	TokenCount    int       `gorm:"default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ClosedAt      *time.Time
}

// AIChatMessage stores individual AI assistant exchanges.
type AIChatMessage struct {
	ID           string    `gorm:"primaryKey;size:36"`
	SessionID    string    `gorm:"size:36;index"`
	Sender       string    `gorm:"size:16"`
	Content      string    `gorm:"type:text"`
	PromptTokens int       `gorm:"default:0"`
	ResultTokens int       `gorm:"default:0"`
	LatencyMS    int       `gorm:"default:0"`
	CreatedAt    time.Time `gorm:"index"`
}
