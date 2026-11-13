package aiagent

import (
	"time"

	"learn-go/internal/domain/identity"
)

// AIUsageLog 记录非对话类 AI 功能（如批改、出题）的 token 用量。
type AIUsageLog struct {
	ID           string        `gorm:"primaryKey;size:36"`
	SchoolID     string        `gorm:"size:36;index"`
	AccountID    string        `gorm:"size:36;index"` // 操作人
	Role         identity.Role `gorm:"size:16"`       // 操作人角色
	Feature      string        `gorm:"size:64;index"` // 例如 "grading"、"question_generation"
	Model        string        `gorm:"size:128"`
	PromptTokens int           `gorm:"default:0"`
	ResultTokens int           `gorm:"default:0"`
	TotalTokens  int           `gorm:"default:0"`
	CreatedAt    time.Time     `gorm:"index"`
}
