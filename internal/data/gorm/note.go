package gormrepo

import (
	"context"
	"gorm.io/gorm"
	notebiz "learn-go/internal/biz/note"
	"time"
)

// NoteStore implements notebiz.NoteRepository with GORM.
type NoteStore struct {
	db *gorm.DB
}

// NewNoteStore creates a new note store instance.
func NewNoteStore(db *gorm.DB) *NoteStore {
	return &NoteStore{db: db}
}

func (s *NoteStore) Create(ctx context.Context, note *notebiz.Note) error {
	return s.db.WithContext(ctx).Create(note).Error
}

func (s *NoteStore) Update(ctx context.Context, note *notebiz.Note) error {
	return s.db.WithContext(ctx).Save(note).Error
}

func (s *NoteStore) FindByID(ctx context.Context, id string) (*notebiz.Note, error) {
	var note notebiz.Note
	if err := s.db.WithContext(ctx).First(&note, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

func (s *NoteStore) ListByOwner(ctx context.Context, ownerID string, includeDeleted bool, status string) ([]notebiz.Note, error) {
	var notes []notebiz.Note
	query := s.db.WithContext(ctx).Where("owner_id = ?", ownerID)
	if !includeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("updated_at DESC").Find(&notes).Error; err != nil {
		return nil, err
	}
	return notes, nil
}

func (s *NoteStore) ListPublishedBySchool(ctx context.Context, schoolID string) ([]notebiz.Note, error) {
	var notes []notebiz.Note
	if err := s.db.WithContext(ctx).
		Where("school_id = ? AND deleted_at IS NULL AND status = ? AND visibility <> ?", schoolID, "published", "private").
		Order("updated_at DESC").
		Find(&notes).Error; err != nil {
		return nil, err
	}
	return notes, nil
}

func (s *NoteStore) SoftDelete(ctx context.Context, id string) error {
	now := time.Now()
	deletedAt := now
	result := s.db.WithContext(ctx).Model(&notebiz.Note{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": &deletedAt,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *NoteStore) Restore(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&notebiz.Note{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": nil,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

var _ notebiz.NoteRepository = (*NoteStore)(nil)
