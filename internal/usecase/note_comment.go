package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/internal/infra/repository"

	"gorm.io/gorm"
)

var (
	// ErrNoteCommentNotAllowed indicates the account has no permission to comment or read.
	ErrNoteCommentNotAllowed = errors.New("not allowed to access note comments")
)

// NoteCommentService manages note comments lifecycle.
type NoteCommentService struct {
	notes    repository.NoteRepository
	comments repository.NoteCommentRepository
	accounts repository.AccountRepository
}

// NewNoteCommentService creates a note comment service instance.
func NewNoteCommentService(notes repository.NoteRepository, comments repository.NoteCommentRepository, accounts repository.AccountRepository) *NoteCommentService {
	return &NoteCommentService{notes: notes, comments: comments, accounts: accounts}
}

// AddComment creates a comment on a note if the account has permission.
func (s *NoteCommentService) AddComment(ctx context.Context, accountID, noteID, content string) (*domain.NoteComment, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	note, err := s.notes.FindByID(ctx, noteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}

	if err := s.ensureCanInteract(account, note); err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("empty comment content")
	}

	now := time.Now()
	comment := &domain.NoteComment{
		ID:         uuid.NewString(),
		NoteID:     note.ID,
		AuthorID:   account.ID,
		AuthorRole: account.Role,
		Content:    content,
		CreatedAt:  now,
	}

	if err := s.comments.Create(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

// ListComments lists comments for a note if the account has permission.
func (s *NoteCommentService) ListComments(ctx context.Context, accountID, noteID string) ([]domain.NoteComment, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	note, err := s.notes.FindByID(ctx, noteID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}

	if err := s.ensureCanInteract(account, note); err != nil {
		return nil, err
	}

	return s.comments.ListByNote(ctx, note.ID)
}

func (s *NoteCommentService) ensureCanInteract(account *domain.Account, note *domain.Note) error {
	if note.OwnerID == account.ID {
		return nil
	}

	if strings.EqualFold(note.Visibility, "private") {
		return ErrNoteCommentNotAllowed
	}

	if account.SchoolID != note.SchoolID {
		return ErrNoteCommentNotAllowed
	}

	if note.Status != "published" {
		return ErrNoteCommentNotAllowed
	}

	return nil
}
