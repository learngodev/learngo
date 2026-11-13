package coursework

import (
	"context"
	"time"

	"learn-go/internal/domain/storage"
)

// AssignmentRepository 负责作业的持久化。
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

// SubmissionRepository 负责学生提交记录的持久化。
type SubmissionRepository interface {
	CreateOrUpdate(ctx context.Context, submission *AssignmentSubmission, items []SubmissionItem) error
	ListByAssignment(ctx context.Context, assignmentID string) ([]AssignmentSubmission, error)
	ListItemsBySubmissionIDs(ctx context.Context, submissionIDs []string) ([]SubmissionItem, error)
	GetByAssignmentAndStudent(ctx context.Context, assignmentID, studentID string) (*AssignmentSubmission, []SubmissionItem, error)
	GetByID(ctx context.Context, submissionID string) (*AssignmentSubmission, []SubmissionItem, error)
	UpdateGrades(ctx context.Context, submission *AssignmentSubmission, items []SubmissionItem) error
	ListByStudentAndAssignments(ctx context.Context, studentID string, assignmentIDs []string) ([]AssignmentSubmission, error)
	StatsByAssignments(ctx context.Context, assignmentIDs []string) ([]AssignmentSubmissionStats, error)
}

// AssignmentSubmissionStats 保存按作业聚合的提交统计数据。
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

// AssignmentScoreDistribution 保存各分数段的人数。
type AssignmentScoreDistribution struct {
	Below60        int64
	Between60And70 int64
	Between70And80 int64
	Between80And90 int64
	Above90        int64
}

// AttachmentRepository 负责作业与提交附件关联的持久化。
type AttachmentRepository interface {
	CreateAssignmentAttachment(ctx context.Context, attachment *AssignmentAttachment) error
	CreateSubmissionAttachment(ctx context.Context, attachment *SubmissionAttachment) error
}

// SubmissionCommentRepository 负责提交批阅评论的持久化。
type SubmissionCommentRepository interface {
	Create(ctx context.Context, comment *SubmissionComment) error
	ListBySubmission(ctx context.Context, submissionID string) ([]SubmissionComment, error)
}
