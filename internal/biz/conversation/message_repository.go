package conversation

import (
	"context"
	"time"
)

// MessageRepository handles chat messages.
type MessageRepository interface {
	Create(ctx context.Context, message *Message) error
	ListByConversation(ctx context.Context, conversationID string, limit int, beforeID string) ([]Message, error)
	GetLastByConversation(ctx context.Context, conversationID string) (*Message, error)
	GetByID(ctx context.Context, id string) (*Message, error)
}

// MessageReceiptRepository records read state for messages.
type MessageReceiptRepository interface {
	CreateBatch(ctx context.Context, receipts []MessageReceipt) error
	CountUnread(ctx context.Context, accountID, conversationID string) (int64, error)
	MarkReadUpTo(ctx context.Context, accountID, conversationID string, ts time.Time) error
}
