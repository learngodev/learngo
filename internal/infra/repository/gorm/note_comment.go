package gormrepo

import (
	"context"

	"learn-go/internal/domain/notebook"

	"gorm.io/gorm"
)

// NoteCommentStore 基于 GORM 实现 notebook.NoteCommentRepository。
type NoteCommentStore struct {
	db *gorm.DB
}

// NewNoteCommentStore 创建笔记评论存储实例。
func NewNoteCommentStore(db *gorm.DB) *NoteCommentStore {
	return &NoteCommentStore{db: db}
}

func (s *NoteCommentStore) Create(ctx context.Context, comment *notebook.NoteComment) error {
	return s.db.WithContext(ctx).Create(comment).Error
}

func (s *NoteCommentStore) ListByNote(ctx context.Context, noteID string) ([]notebook.NoteComment, error) {
	var comments []notebook.NoteComment
	if err := s.db.WithContext(ctx).
		Where("note_id = ?", noteID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

var _ notebook.NoteCommentRepository = (*NoteCommentStore)(nil)
