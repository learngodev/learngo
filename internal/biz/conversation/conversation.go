package conversation

import (
	"time"

	"learn-go/internal/biz/shared"
)

// Conversation represents chat channel.
type Conversation struct {
	ID        string `gorm:"primaryKey;size:36"`
	Type      string `gorm:"size:16"`
	SchoolID  string `gorm:"size:36;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConversationMember ties users to conversations.
type ConversationMember struct {
	ID             string      `gorm:"primaryKey;size:36"`
	ConversationID string      `gorm:"size:36;index"`
	AccountID      string      `gorm:"size:36;index"`
	Role           shared.Role `gorm:"size:16"`
	CreatedAt      time.Time
}
