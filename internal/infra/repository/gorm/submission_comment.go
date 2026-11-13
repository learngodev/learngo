package gormrepo

import (
	"context"

	"learn-go/internal/domain/coursework"

	"gorm.io/gorm"
)

// SubmissionCommentStore 实现 coursework.SubmissionCommentRepository。
type SubmissionCommentStore struct {
	db *gorm.DB
}

// NewSubmissionCommentStore 构造 SubmissionCommentStore 实例。
func NewSubmissionCommentStore(db *gorm.DB) *SubmissionCommentStore {
	return &SubmissionCommentStore{db: db}
}

// Create 持久化一条提交评论。
func (s *SubmissionCommentStore) Create(ctx context.Context, comment *coursework.SubmissionComment) error {
	return s.db.WithContext(ctx).Create(comment).Error
}

// ListBySubmission 按创建时间顺序返回某次提交的评论。
func (s *SubmissionCommentStore) ListBySubmission(ctx context.Context, submissionID string) ([]coursework.SubmissionComment, error) {
	var comments []coursework.SubmissionComment
	if err := s.db.WithContext(ctx).
		Where("submission_id = ?", submissionID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

var _ coursework.SubmissionCommentRepository = (*SubmissionCommentStore)(nil)
