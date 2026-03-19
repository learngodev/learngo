package conversation

import (
	"context"
	"time"
)

// ConversationRepository handles conversation persistence and membership.
type ConversationRepository interface {
	Create(ctx context.Context, conversation *Conversation, members []ConversationMember) error
	GetByID(ctx context.Context, id string) (*Conversation, error)
	ListByAccount(ctx context.Context, accountID string) ([]Conversation, error)
	GetMembers(ctx context.Context, conversationID string) ([]ConversationMember, error)
	IsMember(ctx context.Context, conversationID, accountID string) (bool, error)
	FindDirectBetween(ctx context.Context, schoolID string, participantIDs [2]string) (*Conversation, error)
	UpdateTimestamp(ctx context.Context, conversationID string, ts time.Time) error
}
