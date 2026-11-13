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
	// ErrNoteNotFound 表示笔记不存在。
	ErrNoteNotFound = errors.New("note not found")
	// ErrNoteForbidden 表示当前用户无权执行该操作。
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

// NoteUsecase 管理笔记的生命周期。
type NoteUsecase struct {
	notes    notebook.NoteRepository
	accounts identity.AccountRepository
}

// NewNoteUsecase 创建笔记用例实例。
func NewNoteUsecase(notes notebook.NoteRepository, accounts identity.AccountRepository) *NoteUsecase {
	return &NoteUsecase{notes: notes, accounts: accounts}
}

// CreateNoteInput 描述创建笔记的数据。
type CreateNoteInput struct {
	Title      string
	Content    string
	Visibility string
	Status     string
}

// UpdateNoteInput 描述更新笔记的数据。
type UpdateNoteInput struct {
	Title      string
	Content    string
	Visibility string
	Status     string
}

// CreateNote 为指定账号创建笔记。
func (s *NoteUsecase) CreateNote(ctx context.Context, accountID string, input CreateNoteInput) (*notebook.Note, error) {
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
	note := &notebook.Note{
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

// UpdateNote 更新该账号拥有的笔记。
func (s *NoteUsecase) UpdateNote(ctx context.Context, accountID, noteID string, input UpdateNoteInput) (*notebook.Note, error) {
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

// ListMyNotes 返回所有者的笔记。
func (s *NoteUsecase) ListMyNotes(ctx context.Context, accountID string, status string, includeDeleted bool) ([]notebook.Note, error) {
	status = normalizeStatusFilter(status)
	return s.notes.ListByOwner(ctx, accountID, includeDeleted, status)
}

// ListPublishedNotes 返回该账号所在学校可见的已发布笔记。
func (s *NoteUsecase) ListPublishedNotes(ctx context.Context, accountID string) ([]notebook.Note, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.notes.ListPublishedBySchool(ctx, account.SchoolID)
}

// DeleteNote 将笔记移入回收站。
func (s *NoteUsecase) DeleteNote(ctx context.Context, accountID, noteID string) error {
	if _, err := s.findOwnedNote(ctx, accountID, noteID); err != nil {
		return err
	}
	if err := s.notes.SoftDelete(ctx, noteID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrNoteNotFound
		}
		return err
	}
	return nil
}

// RestoreNote 从回收站恢复笔记。
func (s *NoteUsecase) RestoreNote(ctx context.Context, accountID, noteID string) error {
	note, err := s.findOwnedNote(ctx, accountID, noteID)
	if err != nil {
		return err
	}
	if note.DeletedAt == nil {
		return nil
	}
	if err := s.notes.Restore(ctx, noteID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrNoteNotFound
		}
		return err
	}
	return nil
}

func (s *NoteUsecase) findOwnedNote(ctx context.Context, accountID, noteID string) (*notebook.Note, error) {
	note, err := s.notes.FindByID(ctx, noteID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
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
