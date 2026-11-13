package gormrepo

import (
	"context"

	"learn-go/internal/domain/messaging"

	"gorm.io/gorm"
)

// MessageStore 基于 GORM 实现 messaging.MessageRepository。
type MessageStore struct {
	db *gorm.DB
}

// NewMessageStore 创建消息存储实例。
func NewMessageStore(db *gorm.DB) *MessageStore {
	return &MessageStore{db: db}
}

func (s *MessageStore) Create(ctx context.Context, message *messaging.Message) error {
	return s.db.WithContext(ctx).Create(message).Error
}

func (s *MessageStore) ListByConversation(ctx context.Context, conversationID string, limit int, beforeID string) ([]messaging.Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := s.db.WithContext(ctx).Where("conversation_id = ?", conversationID)

	if beforeID != "" {
		var pivot messaging.Message
		if err := s.db.WithContext(ctx).Select("id", "created_at").First(&pivot, "id = ?", beforeID).Error; err != nil {
			return nil, notFound(err)
		}
		query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", pivot.CreatedAt, pivot.CreatedAt, pivot.ID)
	}

	var messages []messaging.Message
	if err := query.Order("created_at DESC").Limit(limit).Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *MessageStore) GetLastByConversation(ctx context.Context, conversationID string) (*messaging.Message, error) {
	var msg messaging.Message
	if err := s.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		First(&msg).Error; err != nil {
		return nil, notFound(err)
	}
	return &msg, nil
}

func (s *MessageStore) GetByID(ctx context.Context, id string) (*messaging.Message, error) {
	var msg messaging.Message
	if err := s.db.WithContext(ctx).First(&msg, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	return &msg, nil
}

var _ messaging.MessageRepository = (*MessageStore)(nil)
