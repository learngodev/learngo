package conversation

import (
	"time"

	"learn-go/internal/biz/shared"
)

// Message holds chat content.
type Message struct {
	ID             string      `gorm:"primaryKey;size:36"`
	ConversationID string      `gorm:"size:36;index"`
	SenderID       string      `gorm:"size:36;index"`
	SenderRole     shared.Role `gorm:"size:16"`
	Kind           string      `gorm:"size:16"`
	Text           string      `gorm:"type:text"`
	MediaURI       string      `gorm:"size:256"`
	Metadata       string      `gorm:"type:text"`
	CreatedAt      time.Time
}

// MessageReceipt tracks read state.
type MessageReceipt struct {
	ID        string `gorm:"primaryKey;size:36"`
	MessageID string `gorm:"size:36;index"`
	AccountID string `gorm:"size:36;index"`
	ReadAt    *time.Time
	CreatedAt time.Time
}
