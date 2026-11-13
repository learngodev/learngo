package gormrepo

import (
	"context"

	"gorm.io/gorm"

	"learn-go/internal/domain/coursework"
)

// AttachmentStore 基于 GORM 实现 coursework.AttachmentRepository。
type AttachmentStore struct {
	db *gorm.DB
}

// NewAttachmentStore 创建 AttachmentStore。
func NewAttachmentStore(db *gorm.DB) *AttachmentStore {
	return &AttachmentStore{db: db}
}

func (s *AttachmentStore) CreateAssignmentAttachment(ctx context.Context, attachment *coursework.AssignmentAttachment) error {
	return s.db.WithContext(ctx).Create(attachment).Error
}

func (s *AttachmentStore) CreateSubmissionAttachment(ctx context.Context, attachment *coursework.SubmissionAttachment) error {
	return s.db.WithContext(ctx).Create(attachment).Error
}

var _ coursework.AttachmentRepository = (*AttachmentStore)(nil)
