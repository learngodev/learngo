package gormrepo

import (
    "context"
    "time"

    "gorm.io/gorm"

    "learn-go/internal/domain"
    "learn-go/internal/repository"
)

// PasswordResetTokenStore implements repository.PasswordResetTokenRepository.
type PasswordResetTokenStore struct {
    db *gorm.DB
}

// NewPasswordResetTokenStore constructs a PasswordResetTokenStore.
func NewPasswordResetTokenStore(db *gorm.DB) *PasswordResetTokenStore {
    return &PasswordResetTokenStore{db: db}
}

func (s *PasswordResetTokenStore) Create(ctx context.Context, token *domain.PasswordResetToken) error {
    return s.db.WithContext(ctx).Create(token).Error
}

func (s *PasswordResetTokenStore) FindByTokenHash(ctx context.Context, hash string) (*domain.PasswordResetToken, error) {
    var record domain.PasswordResetToken
    if err := s.db.WithContext(ctx).Where("token_hash = ?", hash).First(&record).Error; err != nil {
        return nil, err
    }
    return &record, nil
}

func (s *PasswordResetTokenStore) Consume(ctx context.Context, id string, consumedAt time.Time) error {
    result := s.db.WithContext(ctx).
        Model(&domain.PasswordResetToken{}).
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
        Delete(&domain.PasswordResetToken{}).
        Error
}

var _ repository.PasswordResetTokenRepository = (*PasswordResetTokenStore)(nil)
