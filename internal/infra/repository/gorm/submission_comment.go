package gormrepo

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/infra/repository"

	"gorm.io/gorm"
)

// SubmissionCommentStore implements repository.SubmissionCommentRepository.
type SubmissionCommentStore struct {
	db *gorm.DB
}

// NewSubmissionCommentStore constructs a SubmissionCommentStore instance.
func NewSubmissionCommentStore(db *gorm.DB) *SubmissionCommentStore {
	return &SubmissionCommentStore{db: db}
}

// Create persists a submission comment.
func (s *SubmissionCommentStore) Create(ctx context.Context, comment *domain.SubmissionComment) error {
	return s.db.WithContext(ctx).Create(comment).Error
}

// ListBySubmission returns comments for a submission ordered by creation time.
func (s *SubmissionCommentStore) ListBySubmission(ctx context.Context, submissionID string) ([]domain.SubmissionComment, error) {
	var comments []domain.SubmissionComment
	if err := s.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

var _ repository.SubmissionCommentRepository = (*SubmissionCommentStore)(nil)
