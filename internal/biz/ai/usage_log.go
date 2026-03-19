package ai

import (
	"time"

	"learn-go/internal/biz/shared"
)

// AIUsageLog tracks token usage for non-chat AI features.
type AIUsageLog struct {
	ID           string      `gorm:"primaryKey;size:36"`
	SchoolID     string      `gorm:"size:36;index"`
	AccountID    string      `gorm:"size:36;index"`
	Role         shared.Role `gorm:"size:16"`
	Feature      string      `gorm:"size:64;index"`
	Model        string      `gorm:"size:128"`
	PromptTokens int         `gorm:"default:0"`
	ResultTokens int         `gorm:"default:0"`
	TotalTokens  int         `gorm:"default:0"`
	CreatedAt    time.Time   `gorm:"index"`
}
