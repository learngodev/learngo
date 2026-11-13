package usecase

import (
	"context"
	"errors"
	"learn-go/internal/domain"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain/identity"
	"learn-go/internal/domain/notebook"
)

var (
	// ErrNoteCommentNotAllowed 表示该账号无权评论或查看评论。
	ErrNoteCommentNotAllowed = errors.New("not allowed to access note comments")
)

// NoteCommentUsecase 管理笔记评论的生命周期。
type NoteCommentUsecase struct {
	notes    notebook.NoteRepository
	comments notebook.NoteCommentRepository
	accounts identity.AccountRepository
}

// NewNoteCommentUsecase 创建笔记评论用例实例。
func NewNoteCommentUsecase(notes notebook.NoteRepository, comments notebook.NoteCommentRepository, accounts identity.AccountRepository) *NoteCommentUsecase {
	return &NoteCommentUsecase{notes: notes, comments: comments, accounts: accounts}
}

// AddComment 在账号有权限时为笔记添加评论。
func (s *NoteCommentUsecase) AddComment(ctx context.Context, accountID, noteID, content string) (*notebook.NoteComment, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	note, err := s.notes.FindByID(ctx, noteID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
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
	comment := &notebook.NoteComment{
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

// ListComments 在账号有权限时列出笔记的评论。
func (s *NoteCommentUsecase) ListComments(ctx context.Context, accountID, noteID string) ([]notebook.NoteComment, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	note, err := s.notes.FindByID(ctx, noteID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}

	if err := s.ensureCanInteract(account, note); err != nil {
		return nil, err
	}

	return s.comments.ListByNote(ctx, note.ID)
}

func (s *NoteCommentUsecase) ensureCanInteract(account *identity.Account, note *notebook.Note) error {
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
