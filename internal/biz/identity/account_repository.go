package identity

import (
	"context"
	"time"

	"learn-go/internal/biz/shared"
)

// AccountRepository defines persistence for accounts.
type AccountRepository interface {
	Create(ctx context.Context, account *Account) error
	FindByIdentifier(ctx context.Context, schoolID, identifier string) (*Account, error)
	FindByID(ctx context.Context, id string) (*Account, error)
	ListByIDs(ctx context.Context, ids []string) ([]Account, error)
	ListByRole(ctx context.Context, schoolID string, role shared.Role, status shared.AccountStatus, departmentID string, classID string, courseID string, onlyClassless bool, onlyDepartmentless bool, page int, size int, query string) ([]Account, int64, error)
	UpdateStatus(ctx context.Context, accountID, schoolID string, status shared.AccountStatus) error
	UpdatePasswordHash(ctx context.Context, accountID string, passwordHash string) error
	Update(ctx context.Context, account *Account) error
	Delete(ctx context.Context, accountID, schoolID string) error
}

// PasswordResetTokenRepository manages password reset tokens.
type PasswordResetTokenRepository interface {
	Create(ctx context.Context, token *PasswordResetToken) error
	FindByTokenHash(ctx context.Context, hash string) (*PasswordResetToken, error)
	Consume(ctx context.Context, id string, consumedAt time.Time) error
	DeleteByAccount(ctx context.Context, accountID string) error
}
