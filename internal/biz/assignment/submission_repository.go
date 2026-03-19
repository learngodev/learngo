package assignment

import (
	"context"
)

// SubmissionRepository handles student submissions.
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

// SubmissionCommentRepository handles submission review comments.
type SubmissionCommentRepository interface {
	Create(ctx context.Context, comment *SubmissionComment) error
	ListBySubmission(ctx context.Context, submissionID string) ([]SubmissionComment, error)
}
