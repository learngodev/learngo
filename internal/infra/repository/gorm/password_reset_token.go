package gormrepo

import (
	"context"
	"learn-go/internal/domain"
	"time"

	"gorm.io/gorm"

	"learn-go/internal/domain/identity"
)

// PasswordResetTokenStore 实现 identity.PasswordResetTokenRepository。
type PasswordResetTokenStore struct {
	db *gorm.DB
}

// NewPasswordResetTokenStore 构造 PasswordResetTokenStore。
func NewPasswordResetTokenStore(db *gorm.DB) *PasswordResetTokenStore {
	return &PasswordResetTokenStore{db: db}
}

func (s *PasswordResetTokenStore) Create(ctx context.Context, token *identity.PasswordResetToken) error {
	return s.db.WithContext(ctx).Create(token).Error
}

func (s *PasswordResetTokenStore) FindByTokenHash(ctx context.Context, hash string) (*identity.PasswordResetToken, error) {
	var record identity.PasswordResetToken
	if err := s.db.WithContext(ctx).Where("token_hash = ?", hash).First(&record).Error; err != nil {
		return nil, notFound(err)
	}
	return &record, nil
}

func (s *PasswordResetTokenStore) Consume(ctx context.Context, id string, consumedAt time.Time) error {
	result := s.db.WithContext(ctx).
		Model(&identity.PasswordResetToken{}).
		Where("id = ?", id).
		Updates(map[string]any{"consumed_at": consumedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *PasswordResetTokenStore) DeleteByAccount(ctx context.Context, accountID string) error {
	return s.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Delete(&identity.PasswordResetToken{}).
		Error
}

var _ identity.PasswordResetTokenRepository = (*PasswordResetTokenStore)(nil)
