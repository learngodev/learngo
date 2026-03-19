package gormrepo

import (
	"context"
	"errors"
	"gorm.io/gorm"
	conversationbiz "learn-go/internal/biz/conversation"
)

// MessageStore implements MessageRepository using GORM.
type MessageStore struct {
	db *gorm.DB
}

// NewMessageStore creates a message store instance.
func NewMessageStore(db *gorm.DB) *MessageStore {
	return &MessageStore{db: db}
}

func (s *MessageStore) Create(ctx context.Context, message *conversationbiz.Message) error {
	return s.db.WithContext(ctx).Create(message).Error
}

func (s *MessageStore) ListByConversation(ctx context.Context, conversationID string, limit int, beforeID string) ([]conversationbiz.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := s.db.WithContext(ctx).Where("conversation_id = ?", conversationID)

	if beforeID != "" {
		var pivot conversationbiz.Message
		if err := s.db.WithContext(ctx).Select("id", "created_at").First(&pivot, "id = ?", beforeID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			return nil, err
		}
		query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", pivot.CreatedAt, pivot.CreatedAt, pivot.ID)
	}

	var messages []conversationbiz.Message
	if err := query.Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *MessageStore) GetLastByConversation(ctx context.Context, conversationID string) (*conversationbiz.Message, error) {
	var msg conversationbiz.Message
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		First(&msg).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *MessageStore) GetByID(ctx context.Context, id string) (*conversationbiz.Message, error) {
	var msg conversationbiz.Message
	if err := s.db.WithContext(ctx).First(&msg, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

var _ conversationbiz.MessageRepository = (*MessageStore)(nil)
