package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	identitybiz "learn-go/internal/biz/identity"
	notebiz "learn-go/internal/biz/note"
	"strings"
	"time"
)

var (
	// ErrNoteNotFound indicates the note does not exist.
	ErrNoteNotFound = errors.New("note not found")
	// ErrNoteForbidden indicates the current user cannot perform the operation.
	ErrNoteForbidden = errors.New("not allowed to access note")
)

var (
	allowedVisibilities = map[string]struct{}{
		"private": {},
		"class":   {},
		"school":  {},
	}
	allowedStatuses = map[string]struct{}{
		"draft":     {},
		"published": {},
	}
)

// NoteService manages note lifecycle.
type NoteService struct {
	notes    notebiz.NoteRepository
	accounts identitybiz.AccountRepository
}

// NewNoteService creates a note service instance.
func NewNoteService(notes notebiz.NoteRepository, accounts identitybiz.AccountRepository) *NoteService {
	return &NoteService{notes: notes, accounts: accounts}
}

// CreateNoteInput describes note creation payload.
type CreateNoteInput struct {
	Title      string
	Content    string
	Visibility string
	Status     string
}

// UpdateNoteInput describes note update payload.
type UpdateNoteInput struct {
	Title      string
	Content    string
	Visibility string
	Status     string
}

// CreateNote creates a new note for the given account.
func (s *NoteService) CreateNote(ctx context.Context, accountID string, input CreateNoteInput) (*notebiz.Note, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	visibility := normalizeVisibility(input.Visibility)
	if _, ok := allowedVisibilities[visibility]; !ok {
		return nil, errors.New("invalid visibility")
	}

	status := normalizeStatus(input.Status)
	if _, ok := allowedStatuses[status]; !ok {
		return nil, errors.New("invalid status")
	}

	now := time.Now()
	note := &notebiz.Note{
		ID:         uuid.NewString(),
		SchoolID:   account.SchoolID,
		OwnerID:    account.ID,
		OwnerRole:  account.Role,
		Title:      input.Title,
		Content:    input.Content,
		Visibility: visibility,
		Status:     status,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.notes.Create(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

// UpdateNote updates an existing note owned by the account.
func (s *NoteService) UpdateNote(ctx context.Context, accountID, noteID string, input UpdateNoteInput) (*notebiz.Note, error) {
	note, err := s.findOwnedNote(ctx, accountID, noteID)
	if err != nil {
		return nil, err
	}

	if input.Title != "" {
		note.Title = input.Title
	}
	if input.Content != "" {
		note.Content = input.Content
	}
	if input.Visibility != "" {
		visibility := normalizeVisibility(input.Visibility)
		if _, ok := allowedVisibilities[visibility]; !ok {
			return nil, errors.New("invalid visibility")
		}
		note.Visibility = visibility
	}
	if input.Status != "" {
		status := normalizeStatus(input.Status)
		if _, ok := allowedStatuses[status]; !ok {
			return nil, errors.New("invalid status")
		}
		note.Status = status
	}

	note.UpdatedAt = time.Now()

	if err := s.notes.Update(ctx, note); err != nil {
		return nil, err
	}
	return note, nil
}

// ListMyNotes returns notes for the owner.
func (s *NoteService) ListMyNotes(ctx context.Context, accountID string, status string, includeDeleted bool) ([]notebiz.Note, error) {
	status = normalizeStatusFilter(status)
	return s.notes.ListByOwner(ctx, accountID, includeDeleted, status)
}

// ListPublishedNotes returns published notes visible to the account's school.
func (s *NoteService) ListPublishedNotes(ctx context.Context, accountID string) ([]notebiz.Note, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.notes.ListPublishedBySchool(ctx, account.SchoolID)
}

// DeleteNote moves a note to recycle bin.
func (s *NoteService) DeleteNote(ctx context.Context, accountID, noteID string) error {
	if _, err := s.findOwnedNote(ctx, accountID, noteID); err != nil {
		return err
	}
	if err := s.notes.SoftDelete(ctx, noteID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNoteNotFound
		}
		return err
	}
	return nil
}

// RestoreNote restores a note from recycle bin.
func (s *NoteService) RestoreNote(ctx context.Context, accountID, noteID string) error {
	note, err := s.findOwnedNote(ctx, accountID, noteID)
	if err != nil {
		return err
	}
	if note.DeletedAt == nil {
		return nil
	}
	if err := s.notes.Restore(ctx, noteID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNoteNotFound
		}
		return err
	}
	return nil
}

func (s *NoteService) findOwnedNote(ctx context.Context, accountID, noteID string) (*notebiz.Note, error) {
	note, err := s.notes.FindByID(ctx, noteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}
	if note.OwnerID != accountID {
		return nil, ErrNoteForbidden
	}
	return note, nil
}

func normalizeVisibility(visibility string) string {
	return strings.ToLower(strings.TrimSpace(visibility))
}

func normalizeStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func normalizeStatusFilter(status string) string {
	status = normalizeStatus(status)
	if status == "" {
		return ""
	}
	if _, ok := allowedStatuses[status]; !ok {
		return ""
	}
	return status
}
