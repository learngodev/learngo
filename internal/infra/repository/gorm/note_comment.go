package gormrepo

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/infra/repository"

	"gorm.io/gorm"
)

// NoteCommentStore implements NoteCommentRepository with GORM.
type NoteCommentStore struct {
	db *gorm.DB
}

// NewNoteCommentStore creates a note comment store instance.
func NewNoteCommentStore(db *gorm.DB) *NoteCommentStore {
	return &NoteCommentStore{db: db}
}

func (s *NoteCommentStore) Create(ctx context.Context, comment *domain.NoteComment) error {
	return s.db.WithContext(ctx).Create(comment).Error
}

func (s *NoteCommentStore) ListByNote(ctx context.Context, noteID string) ([]domain.NoteComment, error) {
	var comments []domain.NoteComment
	if err := s.db.WithContext(ctx).
		Where("note_id = ?", noteID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

var _ repository.NoteCommentRepository = (*NoteCommentStore)(nil)
