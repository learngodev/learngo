package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/infra/repository"

	"gorm.io/gorm"
)

// ConversationStore implements ConversationRepository using GORM.
type ConversationStore struct {
	db *gorm.DB
}

// NewConversationStore creates a conversation store instance.
func NewConversationStore(db *gorm.DB) *ConversationStore {
	return &ConversationStore{db: db}
}

func (s *ConversationStore) Create(ctx context.Context, conversation *domain.Conversation, members []domain.ConversationMember) error {
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

func (s *ConversationStore) GetByID(ctx context.Context, id string) (*domain.Conversation, error) {
	var conv domain.Conversation
	if err := s.db.WithContext(ctx).First(&conv, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func (s *ConversationStore) ListByAccount(ctx context.Context, accountID string) ([]domain.Conversation, error) {
	var convs []domain.Conversation
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

func (s *ConversationStore) GetMembers(ctx context.Context, conversationID string) ([]domain.ConversationMember, error) {
	var members []domain.ConversationMember
	if err := s.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (s *ConversationStore) IsMember(ctx context.Context, conversationID, accountID string) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&domain.ConversationMember{}).
		Where("conversation_id = ? AND account_id = ?", conversationID, accountID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *ConversationStore) FindDirectBetween(ctx context.Context, schoolID string, participantIDs [2]string) (*domain.Conversation, error) {
	var conv domain.Conversation

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
	result := s.db.WithContext(ctx).Model(&domain.Conversation{}).
		Where("id = ?", conversationID).
		Updates(map[string]interface{}{"updated_at": ts})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

var _ repository.ConversationRepository = (*ConversationStore)(nil)
