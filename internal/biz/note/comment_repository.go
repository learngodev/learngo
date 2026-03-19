package note

import "context"

// NoteCommentRepository handles note comment persistence.
type NoteCommentRepository interface {
	Create(ctx context.Context, comment *NoteComment) error
	ListByNote(ctx context.Context, noteID string) ([]NoteComment, error)
}
