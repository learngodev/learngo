package gormrepo

import (
	"context"
	"learn-go/internal/domain"
	"time"

	"learn-go/internal/domain/messaging"

	"gorm.io/gorm"
)

// ConversationStore 基于 GORM 实现 messaging.ConversationRepository。
type ConversationStore struct {
	db *gorm.DB
}

// NewConversationStore 创建会话存储实例。
func NewConversationStore(db *gorm.DB) *ConversationStore {
	return &ConversationStore{db: db}
}

func (s *ConversationStore) Create(ctx context.Context, conversation *messaging.Conversation, members []messaging.ConversationMember) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conversation).Error; err != nil {
			return err
		}
		if len(members) > 0 {
			if err := tx.Create(&members).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ConversationStore) GetByID(ctx context.Context, id string) (*messaging.Conversation, error) {
	var conv messaging.Conversation
	if err := s.db.WithContext(ctx).First(&conv, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	return &conv, nil
}

func (s *ConversationStore) ListByAccount(ctx context.Context, accountID string) ([]messaging.Conversation, error) {
	var convs []messaging.Conversation
	err := s.db.WithContext(ctx).
		Joins("JOIN conversation_members cm ON cm.conversation_id = conversations.id").
		Where("cm.account_id = ?", accountID).
		Order("conversations.updated_at DESC").
		Find(&convs).Error
	if err != nil {
		return nil, err
	}
	return convs, nil
}

func (s *ConversationStore) GetMembers(ctx context.Context, conversationID string) ([]messaging.ConversationMember, error) {
	var members []messaging.ConversationMember
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (s *ConversationStore) IsMember(ctx context.Context, conversationID, accountID string) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&messaging.ConversationMember{}).
		Where("conversation_id = ? AND account_id = ?", conversationID, accountID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *ConversationStore) FindDirectBetween(ctx context.Context, schoolID string, participantIDs [2]string) (*messaging.Conversation, error) {
	var conv messaging.Conversation

	err := s.db.WithContext(ctx).
		Joins("JOIN conversation_members cm ON cm.conversation_id = conversations.id").
		Where("conversations.school_id = ? AND conversations.type = ?", schoolID, "direct").
		Where("cm.account_id IN ?", participantIDs[:]).
		Group("conversations.id").
		Having("COUNT(DISTINCT cm.account_id) = ?", len(participantIDs)).
		Having("COUNT(*) = ?", len(participantIDs)).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (s *ConversationStore) UpdateTimestamp(ctx context.Context, conversationID string, ts time.Time) error {
	result := s.db.WithContext(ctx).Model(&messaging.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{"updated_at": ts})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

var _ messaging.ConversationRepository = (*ConversationStore)(nil)
