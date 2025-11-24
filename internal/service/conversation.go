package service

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

var (
	// ErrConversationNotFound indicates the conversation does not exist or accessible.
	ErrConversationNotFound = errors.New("conversation not found")
	// ErrConversationForbidden indicates the account is not a member of the conversation.
	ErrConversationForbidden = errors.New("conversation forbidden")
	// ErrConversationInvalid indicates invalid parameters when creating conversations.
	ErrConversationInvalid = errors.New("invalid conversation request")
)

// ConversationService handles IM related operations.
type ConversationService struct {
	conversations repository.ConversationRepository
	messages      repository.MessageRepository
	receipts      repository.MessageReceiptRepository
	accounts      repository.AccountRepository
}

// NewConversationService constructs a ConversationService instance.
func NewConversationService(conversations repository.ConversationRepository, messages repository.MessageRepository, receipts repository.MessageReceiptRepository, accounts repository.AccountRepository) *ConversationService {
	return &ConversationService{
		conversations: conversations,
		messages:      messages,
		receipts:      receipts,
		accounts:      accounts,
	}
}

// ConversationSummary aggregates conversation with metadata.
type ConversationSummary struct {
	Conversation   domain.Conversation
	Members        []domain.ConversationMember
	MemberProfiles map[string]domain.Account
	LastMessage    *domain.Message
	UnreadCount    int64
}

// SendMessageInput describes payload for sending a message.
type SendMessageInput struct {
	ConversationID string
	Kind           string
	Text           string
	MediaURI       string
	Metadata       string
}

// CreateDirectConversation ensures a direct conversation exists between two accounts.
func (s *ConversationService) CreateDirectConversation(ctx context.Context, initiatorID, participantID string) (*ConversationSummary, error) {
	if initiatorID == participantID {
		return nil, ErrConversationInvalid
	}

	initiator, err := s.accounts.FindByID(ctx, initiatorID)
	if err != nil {
		return nil, err
	}
	participant, err := s.accounts.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}
	if initiator.SchoolID != participant.SchoolID {
		return nil, ErrConversationInvalid
	}

	ids := [2]string{initiatorID, participantID}
	sort.Strings(ids[:])

	conv, err := s.conversations.FindDirectBetween(ctx, initiator.SchoolID, ids)
	if err == nil {
		members, err := s.conversations.GetMembers(ctx, conv.ID)
		if err != nil {
			return nil, err
		}
		var last *domain.Message
		lastMsg, err := s.messages.GetLastByConversation(ctx, conv.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if err == nil {
			last = lastMsg
		}
		unread, err := s.receipts.CountUnread(ctx, initiatorID, conv.ID)
		if err != nil {
			return nil, err
		}
		return &ConversationSummary{Conversation: *conv, Members: members, LastMessage: last, UnreadCount: unread}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	conv = &domain.Conversation{
		ID:        uuid.NewString(),
		Type:      "direct",
		SchoolID:  initiator.SchoolID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	members := []domain.ConversationMember{
		{
			ID:             uuid.NewString(),
			ConversationID: conv.ID,
			AccountID:      initiator.ID,
			Role:           initiator.Role,
			CreatedAt:      now,
		},
		{
			ID:             uuid.NewString(),
			ConversationID: conv.ID,
			AccountID:      participant.ID,
			Role:           participant.Role,
			CreatedAt:      now,
		},
	}

	if err := s.conversations.Create(ctx, conv, members); err != nil {
		return nil, err
	}

	return &ConversationSummary{Conversation: *conv, Members: members, UnreadCount: 0}, nil
}

// ListConversations returns conversations for an account.
func (s *ConversationService) ListConversations(ctx context.Context, accountID string) ([]ConversationSummary, error) {
	convs, err := s.conversations.ListByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	summaries := make([]ConversationSummary, 0, len(convs))
	for _, conv := range convs {
		members, err := s.conversations.GetMembers(ctx, conv.ID)
		if err != nil {
			return nil, err
		}
		var last *domain.Message
		lastMsg, err := s.messages.GetLastByConversation(ctx, conv.ID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
		} else {
			last = lastMsg
		}
		unread, err := s.receipts.CountUnread(ctx, accountID, conv.ID)
		if err != nil {
			return nil, err
		}

		memberIDs := make([]string, len(members))
		for i, m := range members {
			memberIDs[i] = m.AccountID
		}
		accounts, err := s.accounts.ListByIDs(ctx, memberIDs)
		if err != nil {
			return nil, err
		}
		profiles := make(map[string]domain.Account)
		for _, acc := range accounts {
			profiles[acc.ID] = acc
		}

		summaries = append(summaries, ConversationSummary{
			Conversation:   conv,
			Members:        members,
			MemberProfiles: profiles,
			LastMessage:    last,
			UnreadCount:    unread,
		})
	}
	return summaries, nil
}

// SendMessage stores a message in a conversation.
func (s *ConversationService) SendMessage(ctx context.Context, senderID string, input SendMessageInput) (*domain.Message, error) {
	ok, err := s.conversations.IsMember(ctx, input.ConversationID, senderID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConversationForbidden
	}

	sender, err := s.accounts.FindByID(ctx, senderID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	msg := &domain.Message{
		ID:             uuid.NewString(),
		ConversationID: input.ConversationID,
		SenderID:       sender.ID,
		SenderRole:     sender.Role,
		Kind:           input.Kind,
		Text:           input.Text,
		MediaURI:       input.MediaURI,
		Metadata:       input.Metadata,
		CreatedAt:      now,
	}

	if err := s.messages.Create(ctx, msg); err != nil {
		return nil, err
	}
	if err := s.conversations.UpdateTimestamp(ctx, input.ConversationID, now); err != nil {
		return nil, err
	}

	members, err := s.conversations.GetMembers(ctx, input.ConversationID)
	if err != nil {
		return nil, err
	}

	receipts := make([]domain.MessageReceipt, 0, len(members))
	for _, member := range members {
		if member.AccountID == senderID {
			continue
		}
		receipts = append(receipts, domain.MessageReceipt{
			ID:        uuid.NewString(),
			MessageID: msg.ID,
			AccountID: member.AccountID,
			CreatedAt: now,
		})
	}

	if len(receipts) > 0 {
		if err := s.receipts.CreateBatch(ctx, receipts); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

// ListMessages returns messages in a conversation respecting membership.
func (s *ConversationService) ListMessages(ctx context.Context, accountID, conversationID string, limit int, beforeID string) ([]domain.Message, error) {
	ok, err := s.conversations.IsMember(ctx, conversationID, accountID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConversationForbidden
	}

	messages, err := s.messages.ListByConversation(ctx, conversationID, limit, beforeID)
	if err != nil {
		return nil, err
	}

	// Results are descending by default; flip to ascending for clients.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetConversationSummary returns metadata for a conversation the account participates in.
func (s *ConversationService) GetConversationSummary(ctx context.Context, accountID, conversationID string) (*ConversationSummary, error) {
	ok, err := s.conversations.IsMember(ctx, conversationID, accountID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConversationForbidden
	}

	conv, err := s.conversations.GetByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	members, err := s.conversations.GetMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	var last *domain.Message
	lastMsg, err := s.messages.GetLastByConversation(ctx, conversationID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		last = lastMsg
	}

	unread, err := s.receipts.CountUnread(ctx, accountID, conversationID)
	if err != nil {
		return nil, err
	}

	return &ConversationSummary{
		Conversation: *conv,
		Members:      members,
		LastMessage:  last,
		UnreadCount:  unread,
	}, nil
}

// MarkRead marks messages as read up to the provided message.
func (s *ConversationService) MarkRead(ctx context.Context, accountID, conversationID, messageID string) error {
	ok, err := s.conversations.IsMember(ctx, conversationID, accountID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrConversationForbidden
	}

	msg, err := s.messages.GetByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrConversationNotFound
		}
		return err
	}
	if msg.ConversationID != conversationID {
		return ErrConversationNotFound
	}

	if err := s.receipts.MarkReadUpTo(ctx, accountID, conversationID, msg.CreatedAt); err != nil {
		return err
	}
	return nil
}

// IsMember returns whether the account participates in the conversation.
func (s *ConversationService) IsMember(ctx context.Context, accountID, conversationID string) (bool, error) {
	return s.conversations.IsMember(ctx, conversationID, accountID)
}
