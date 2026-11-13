package identity

import (
	"time"
)

// PasswordResetToken 存储用于重置账号密码的一次性令牌。
type PasswordResetToken struct {
	ID         string    `gorm:"primaryKey;size:36"`
	AccountID  string    `gorm:"size:36;index"`
	SchoolID   string    `gorm:"size:36;index"`
	TokenHash  string    `gorm:"size:128;uniqueIndex"`
	ExpiresAt  time.Time `gorm:"index"`
	ConsumedAt *time.Time
	CreatedAt  time.Time
}
