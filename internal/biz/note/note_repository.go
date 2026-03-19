package note

import "context"

// NoteRepository handles note persistence.
type NoteRepository interface {
	Create(ctx context.Context, note *Note) error
	Update(ctx context.Context, note *Note) error
	FindByID(ctx context.Context, id string) (*Note, error)
	ListByOwner(ctx context.Context, ownerID string, includeDeleted bool, status string) ([]Note, error)
	ListPublishedBySchool(ctx context.Context, schoolID string) ([]Note, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}
