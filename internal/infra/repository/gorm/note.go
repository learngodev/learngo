package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/infra/repository"

	"gorm.io/gorm"
)

// NoteStore implements repository.NoteRepository with GORM.
type NoteStore struct {
	db *gorm.DB
}

// NewNoteStore creates a new note store instance.
func NewNoteStore(db *gorm.DB) *NoteStore {
	return &NoteStore{db: db}
}

func (s *NoteStore) Create(ctx context.Context, note *domain.Note) error {
	return s.db.WithContext(ctx).Create(note).Error
}

func (s *NoteStore) Update(ctx context.Context, note *domain.Note) error {
	return s.db.WithContext(ctx).Save(note).Error
}

func (s *NoteStore) FindByID(ctx context.Context, id string) (*domain.Note, error) {
	var note domain.Note
	if err := s.db.WithContext(ctx).First(&note, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &note, nil
}

func (s *NoteStore) ListByOwner(ctx context.Context, ownerID string, includeDeleted bool, status string) ([]domain.Note, error) {
	var notes []domain.Note
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

func (s *NoteStore) ListPublishedBySchool(ctx context.Context, schoolID string) ([]domain.Note, error) {
	var notes []domain.Note
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
	result := s.db.WithContext(ctx).Model(&domain.Note{}).
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
	result := s.db.WithContext(ctx).Model(&domain.Note{}).
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

var _ repository.NoteRepository = (*NoteStore)(nil)
