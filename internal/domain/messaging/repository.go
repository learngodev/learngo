package messaging

import (
	"context"
	"time"
)

// ConversationRepository 负责会话及其成员的持久化。
type ConversationRepository interface {
	Create(ctx context.Context, conversation *Conversation, members []ConversationMember) error
	GetByID(ctx context.Context, id string) (*Conversation, error)
	ListByAccount(ctx context.Context, accountID string) ([]Conversation, error)
	GetMembers(ctx context.Context, conversationID string) ([]ConversationMember, error)
	IsMember(ctx context.Context, conversationID, accountID string) (bool, error)
	FindDirectBetween(ctx context.Context, schoolID string, participantIDs [2]string) (*Conversation, error)
	UpdateTimestamp(ctx context.Context, conversationID string, ts time.Time) error
}

// MessageRepository 负责聊天消息的持久化。
type MessageRepository interface {
	Create(ctx context.Context, message *Message) error
	ListByConversation(ctx context.Context, conversationID string, limit int, beforeID string) ([]Message, error)
	GetLastByConversation(ctx context.Context, conversationID string) (*Message, error)
	GetByID(ctx context.Context, id string) (*Message, error)
}

// MessageReceiptRepository 记录消息的已读状态。
type MessageReceiptRepository interface {
	CreateBatch(ctx context.Context, receipts []MessageReceipt) error
	CountUnread(ctx context.Context, accountID, conversationID string) (int64, error)
	MarkReadUpTo(ctx context.Context, accountID, conversationID string, ts time.Time) error
}
