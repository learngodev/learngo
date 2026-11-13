package messaging

import (
	"time"

	"learn-go/internal/domain/identity"
)

// Message 保存聊天内容。
type Message struct {
	ID             string        `gorm:"primaryKey;size:36"`
	ConversationID string        `gorm:"size:36;index"`
	SenderID       string        `gorm:"size:36;index"`
	SenderRole     identity.Role `gorm:"size:16"`
	Kind           string        `gorm:"size:16"` // text、image、video、audio
	Text           string        `gorm:"type:text"`
	MediaURI       string        `gorm:"size:256"`
	Metadata       string        `gorm:"type:text"`
	CreatedAt      time.Time
}

// MessageReceipt 记录已读状态。
type MessageReceipt struct {
	ID        string `gorm:"primaryKey;size:36"`
	MessageID string `gorm:"size:36;index"`
	AccountID string `gorm:"size:36;index"`
	ReadAt    *time.Time
	CreatedAt time.Time
}
