package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain/messaging"

	"gorm.io/gorm"
)

// MessageReceiptStore 基于 GORM 实现 messaging.MessageReceiptRepository。
type MessageReceiptStore struct {
	db *gorm.DB
}

// NewMessageReceiptStore 创建消息回执存储实例。
func NewMessageReceiptStore(db *gorm.DB) *MessageReceiptStore {
	return &MessageReceiptStore{db: db}
}

func (s *MessageReceiptStore) CreateBatch(ctx context.Context, receipts []messaging.MessageReceipt) error {
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
	sub := s.db.WithContext(ctx).Model(&messaging.Message{}).
		Select("id").
		Where("conversation_id = ? AND created_at <= ?", conversationID, ts)

	return s.db.WithContext(ctx).Model(&messaging.MessageReceipt{}).
		Where("account_id = ? AND read_at IS NULL", accountID).
		Where("message_id IN (?)", sub).
		Updates(map[string]interface{}{"read_at": now}).Error
}

var _ messaging.MessageReceiptRepository = (*MessageReceiptStore)(nil)
