package messaging

import (
	"time"

	"learn-go/internal/domain/identity"
)

// Conversation 表示一个聊天会话。
type Conversation struct {
	ID        string `gorm:"primaryKey;size:36"`
	Type      string `gorm:"size:16"` // direct（单聊）或 group（群聊）
	SchoolID  string `gorm:"size:36;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConversationMember 关联用户与会话。
type ConversationMember struct {
	ID             string        `gorm:"primaryKey;size:36"`
	ConversationID string        `gorm:"size:36;index"`
	AccountID      string        `gorm:"size:36;index"`
	Role           identity.Role `gorm:"size:16"`
	CreatedAt      time.Time
}
