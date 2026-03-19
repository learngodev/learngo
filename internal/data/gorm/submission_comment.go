package gormrepo

import (
	"context"
	"gorm.io/gorm"
	assignmentbiz "learn-go/internal/biz/assignment"
)

// SubmissionCommentStore implements assignmentbiz.SubmissionCommentRepository.
type SubmissionCommentStore struct {
	db *gorm.DB
}

// NewSubmissionCommentStore constructs a SubmissionCommentStore instance.
func NewSubmissionCommentStore(db *gorm.DB) *SubmissionCommentStore {
	return &SubmissionCommentStore{db: db}
}

// Create persists a submission comment.
func (s *SubmissionCommentStore) Create(ctx context.Context, comment *assignmentbiz.SubmissionComment) error {
	return s.db.WithContext(ctx).Create(comment).Error
}

// ListBySubmission returns comments for a submission ordered by creation time.
func (s *SubmissionCommentStore) ListBySubmission(ctx context.Context, submissionID string) ([]assignmentbiz.SubmissionComment, error) {
	var comments []assignmentbiz.SubmissionComment
	if err := s.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

var _ assignmentbiz.SubmissionCommentRepository = (*SubmissionCommentStore)(nil)
