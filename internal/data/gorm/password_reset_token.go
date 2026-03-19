package gormrepo

import (
	"context"
	"gorm.io/gorm"
	identitybiz "learn-go/internal/biz/identity"
	"time"
)

// PasswordResetTokenStore implements identitybiz.PasswordResetTokenRepository.
type PasswordResetTokenStore struct {
	db *gorm.DB
}

// NewPasswordResetTokenStore constructs a PasswordResetTokenStore.
func NewPasswordResetTokenStore(db *gorm.DB) *PasswordResetTokenStore {
	return &PasswordResetTokenStore{db: db}
}

func (s *PasswordResetTokenStore) Create(ctx context.Context, token *identitybiz.PasswordResetToken) error {
	return s.db.WithContext(ctx).Create(token).Error
}

func (s *PasswordResetTokenStore) FindByTokenHash(ctx context.Context, hash string) (*identitybiz.PasswordResetToken, error) {
	var record identitybiz.PasswordResetToken
	if err := s.db.WithContext(ctx).Where("token_hash = ?", hash).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *PasswordResetTokenStore) Consume(ctx context.Context, id string, consumedAt time.Time) error {
	result := s.db.WithContext(ctx).
		Model(&identitybiz.PasswordResetToken{}).
		Where("id = ?", id).
		Updates(map[string]any{"consumed_at": consumedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *PasswordResetTokenStore) DeleteByAccount(ctx context.Context, accountID string) error {
	return s.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Delete(&identitybiz.PasswordResetToken{}).
		Error
}

var _ identitybiz.PasswordResetTokenRepository = (*PasswordResetTokenStore)(nil)
