package gormrepo

import (
	"context"
	"gorm.io/gorm"
	conversationbiz "learn-go/internal/biz/conversation"
	"time"
)

// MessageReceiptStore implements MessageReceiptRepository using GORM.
type MessageReceiptStore struct {
	db *gorm.DB
}

// NewMessageReceiptStore creates a message receipt store instance.
func NewMessageReceiptStore(db *gorm.DB) *MessageReceiptStore {
	return &MessageReceiptStore{db: db}
}

func (s *MessageReceiptStore) CreateBatch(ctx context.Context, receipts []conversationbiz.MessageReceipt) error {
	if len(receipts) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Create(&receipts).Error
}

func (s *MessageReceiptStore) CountUnread(ctx context.Context, accountID, conversationID string) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("message_receipts mr").
		Joins("JOIN messages m ON m.id = mr.message_id").
		Where("mr.account_id = ? AND mr.read_at IS NULL AND m.conversation_id = ?", accountID, conversationID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *MessageReceiptStore) MarkReadUpTo(ctx context.Context, accountID, conversationID string, ts time.Time) error {
	now := time.Now()
	sub := s.db.WithContext(ctx).Model(&conversationbiz.Message{}).
		Select("id").
		Where("conversation_id = ? AND created_at <= ?", conversationID, ts)

	return s.db.WithContext(ctx).Model(&conversationbiz.MessageReceipt{}).
		Where("account_id = ? AND read_at IS NULL", accountID).
		Where("message_id IN (?)", sub).
		Updates(map[string]interface{}{"read_at": now}).Error
}

var _ conversationbiz.MessageReceiptRepository = (*MessageReceiptStore)(nil)
