package notebook

import (
	"context"
)

// NoteRepository 负责笔记的持久化。
type NoteRepository interface {
	Create(ctx context.Context, note *Note) error
	Update(ctx context.Context, note *Note) error
	FindByID(ctx context.Context, id string) (*Note, error)
	ListByOwner(ctx context.Context, ownerID string, includeDeleted bool, status string) ([]Note, error)
	ListPublishedBySchool(ctx context.Context, schoolID string) ([]Note, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}

// NoteCommentRepository 负责笔记评论的持久化。
type NoteCommentRepository interface {
	Create(ctx context.Context, comment *NoteComment) error
	ListByNote(ctx context.Context, noteID string) ([]NoteComment, error)
}
