package identity

import (
	"time"

	"gorm.io/gorm"
)

// Role 定义平台角色。
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

// AccountStatus 表示账号的可用状态。
type AccountStatus string

const (
	AccountStatusActive                AccountStatus = "active"
	AccountStatusLocked                AccountStatus = "locked"
	AccountStatusPasswordResetRequired AccountStatus = "password_reset_required"
)

// Account 表示与用户类型绑定的登录凭据。
type Account struct {
	ID           string        `gorm:"primaryKey;size:36"`
	SchoolID     string        `gorm:"size:36;index"`
	Role         Role          `gorm:"size:16;index"`
	Status       AccountStatus `gorm:"size:32;index;default:'active'"`
	Identifier   string        `gorm:"size:64;uniqueIndex:idx_accounts_identifier,where:deleted_at IS NULL"` // 教师工号或学生学号
	PasswordHash string        `gorm:"size:128"`
	DisplayName  string        `gorm:"size:128"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
