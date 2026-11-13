package usecase

import (
	"context"
	"errors"
	"learn-go/internal/domain"
	"sort"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain/identity"
	"learn-go/internal/domain/messaging"
)

var (
	// ErrConversationNotFound 表示会话不存在或不可访问。
	ErrConversationNotFound = errors.New("conversation not found")
	// ErrConversationForbidden 表示该账号不是会话成员。
	ErrConversationForbidden = errors.New("conversation forbidden")
	// ErrConversationInvalid 表示创建会话时参数无效。
	ErrConversationInvalid = errors.New("invalid conversation request")
)

// ConversationUsecase 负责即时通讯相关操作。
type ConversationUsecase struct {
	conversations messaging.ConversationRepository
	messages      messaging.MessageRepository
	receipts      messaging.MessageReceiptRepository
	accounts      identity.AccountRepository
}

// NewConversationUsecase 构造 ConversationUsecase 实例。
func NewConversationUsecase(conversations messaging.ConversationRepository, messages messaging.MessageRepository, receipts messaging.MessageReceiptRepository, accounts identity.AccountRepository) *ConversationUsecase {
	return &ConversationUsecase{
		conversations: conversations,
		messages:      messages,
		receipts:      receipts,
		accounts:      accounts,
	}
}

// ConversationSummary 聚合会话及其元数据。
type ConversationSummary struct {
	Conversation   messaging.Conversation
	Members        []messaging.ConversationMember
	MemberProfiles map[string]identity.Account
	LastMessage    *messaging.Message
	UnreadCount    int64
}

// SendMessageInput 描述发送消息的数据。
type SendMessageInput struct {
	ConversationID string
	Kind           string
	Text           string
	MediaURI       string
	Metadata       string
}

// CreateDirectConversation 确保两个账号之间存在单聊会话。
func (s *ConversationUsecase) CreateDirectConversation(ctx context.Context, initiatorID, participantID string) (*ConversationSummary, error) {
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
		var last *messaging.Message
		lastMsg, err := s.messages.GetLastByConversation(ctx, conv.ID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
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
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	now := time.Now()
	conv = &messaging.Conversation{
		ID:        uuid.NewString(),
		Type:      "direct",
		SchoolID:  initiator.SchoolID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	members := []messaging.ConversationMember{
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

// ListConversations 返回某账号的会话列表。
func (s *ConversationUsecase) ListConversations(ctx context.Context, accountID string) ([]ConversationSummary, error) {
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
		var last *messaging.Message
		lastMsg, err := s.messages.GetLastByConversation(ctx, conv.ID)
		if err != nil {
			if !errors.Is(err, domain.ErrNotFound) {
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
		profiles := make(map[string]identity.Account)
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

// SendMessage 在会话中保存一条消息。
func (s *ConversationUsecase) SendMessage(ctx context.Context, senderID string, input SendMessageInput) (*messaging.Message, error) {
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
	msg := &messaging.Message{
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

	receipts := make([]messaging.MessageReceipt, 0, len(members))
	for _, member := range members {
		if member.AccountID == senderID {
			continue
		}
		receipts = append(receipts, messaging.MessageReceipt{
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

// ListMessages 在校验成员身份后返回会话中的消息。
func (s *ConversationUsecase) ListMessages(ctx context.Context, accountID, conversationID string, limit int, beforeID string) ([]messaging.Message, error) {
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

	// 查询结果默认倒序，返回给客户端前翻转为正序。
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetConversationSummary 返回账号所参与会话的元数据。
func (s *ConversationUsecase) GetConversationSummary(ctx context.Context, accountID, conversationID string) (*ConversationSummary, error) {
	ok, err := s.conversations.IsMember(ctx, conversationID, accountID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConversationForbidden
	}

	conv, err := s.conversations.GetByID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}

	members, err := s.conversations.GetMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	var last *messaging.Message
	lastMsg, err := s.messages.GetLastByConversation(ctx, conversationID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
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

// MarkRead 将截至指定消息的所有消息标记为已读。
func (s *ConversationUsecase) MarkRead(ctx context.Context, accountID, conversationID, messageID string) error {
	ok, err := s.conversations.IsMember(ctx, conversationID, accountID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrConversationForbidden
	}

	msg, err := s.messages.GetByID(ctx, messageID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
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

// IsMember 返回账号是否为会话成员。
func (s *ConversationUsecase) IsMember(ctx context.Context, accountID, conversationID string) (bool, error) {
	return s.conversations.IsMember(ctx, conversationID, accountID)
}
