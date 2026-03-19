package identity

import (
	"time"

	"gorm.io/gorm"

	"learn-go/internal/biz/shared"
)

// Account represents login credentials tied to a user type.
type Account struct {
	ID           string               `gorm:"primaryKey;size:36"`
	SchoolID     string               `gorm:"size:36;index"`
	Role         shared.Role          `gorm:"size:16;index"`
	Status       shared.AccountStatus `gorm:"size:32;index;default:'active'"`
	Identifier   string               `gorm:"size:64;uniqueIndex:idx_accounts_identifier,where:deleted_at IS NULL"`
	PasswordHash string               `gorm:"size:128"`
	DisplayName  string               `gorm:"size:128"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// PasswordResetToken stores a single-use token for resetting account passwords.
type PasswordResetToken struct {
	ID         string    `gorm:"primaryKey;size:36"`
	AccountID  string    `gorm:"size:36;index"`
	SchoolID   string    `gorm:"size:36;index"`
	TokenHash  string    `gorm:"size:128;uniqueIndex"`
	ExpiresAt  time.Time `gorm:"index"`
	ConsumedAt *time.Time
	CreatedAt  time.Time
}
