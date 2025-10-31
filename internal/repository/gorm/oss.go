package gormrepo

import (
    "context"

    "learn-go/internal/domain"
    "learn-go/internal/repository"

    "gorm.io/gorm"
)

// OssCredentialStore implements repository.OssCredentialRepository using GORM.
type OssCredentialStore struct {
    db *gorm.DB
}

// NewOssCredentialStore creates a new OssCredentialStore.
func NewOssCredentialStore(db *gorm.DB) *OssCredentialStore {
    return &OssCredentialStore{db: db}
}

func (s *OssCredentialStore) List(ctx context.Context, schoolID string) ([]domain.OssCredential, error) {
    var out []domain.OssCredential
    if err := s.db.WithContext(ctx).
        Where("school_id = ?", schoolID).
        Order("is_primary DESC, created_at DESC").
        Find(&out).Error; err != nil {
        return nil, err
    }
    return out, nil
}

func (s *OssCredentialStore) GetByID(ctx context.Context, credentialID, schoolID string) (*domain.OssCredential, error) {
    var credential domain.OssCredential
    if err := s.db.WithContext(ctx).
        First(&credential, "id = ? AND school_id = ?", credentialID, schoolID).Error; err != nil {
        return nil, err
    }
    return &credential, nil
}

func (s *OssCredentialStore) Update(ctx context.Context, credentialID, schoolID string, updates map[string]any) (*domain.OssCredential, error) {
    if len(updates) == 0 {
        return s.GetByID(ctx, credentialID, schoolID)
    }
    result := s.db.WithContext(ctx).
        Model(&domain.OssCredential{}).
        Where("id = ? AND school_id = ?", credentialID, schoolID).
        Updates(updates)
    if result.Error != nil {
        return nil, result.Error
    }
    if result.RowsAffected == 0 {
        return nil, gorm.ErrRecordNotFound
    }
    return s.GetByID(ctx, credentialID, schoolID)
}

func (s *OssCredentialStore) SetPrimary(ctx context.Context, credentialID, schoolID string) (*domain.OssCredential, error) {
    var updated *domain.OssCredential
    err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Model(&domain.OssCredential{}).
            Where("school_id = ?", schoolID).
            Updates(map[string]any{"is_primary": false}).Error; err != nil {
            return err
        }
        result := tx.Model(&domain.OssCredential{}).
            Where("id = ? AND school_id = ?", credentialID, schoolID).
            Updates(map[string]any{"is_primary": true, "active": true})
        if result.Error != nil {
            return result.Error
        }
        if result.RowsAffected == 0 {
            return gorm.ErrRecordNotFound
        }
        var cred domain.OssCredential
        if err := tx.First(&cred, "id = ? AND school_id = ?", credentialID, schoolID).Error; err != nil {
            return err
        }
        updated = &cred
        return nil
    })
    if err != nil {
        return nil, err
    }
    return updated, nil
}

var _ repository.OssCredentialRepository = (*OssCredentialStore)(nil)

// OssPolicyStore implements repository.OssPolicyRepository using GORM.
type OssPolicyStore struct {
    db *gorm.DB
}

// NewOssPolicyStore creates a new OssPolicyStore.
func NewOssPolicyStore(db *gorm.DB) *OssPolicyStore {
    return &OssPolicyStore{db: db}
}

func (s *OssPolicyStore) List(ctx context.Context, schoolID string) ([]domain.OssPolicy, error) {
    var out []domain.OssPolicy
    if err := s.db.WithContext(ctx).
        Where("school_id = ?", schoolID).
        Order("created_at ASC").
        Find(&out).Error; err != nil {
        return nil, err
    }
    return out, nil
}

func (s *OssPolicyStore) GetByID(ctx context.Context, policyID, schoolID string) (*domain.OssPolicy, error) {
    var policy domain.OssPolicy
    if err := s.db.WithContext(ctx).
        First(&policy, "id = ? AND school_id = ?", policyID, schoolID).Error; err != nil {
        return nil, err
    }
    return &policy, nil
}

func (s *OssPolicyStore) UpdateStatus(ctx context.Context, policyID, schoolID string, status domain.OssPolicyStatus) (*domain.OssPolicy, error) {
    result := s.db.WithContext(ctx).
        Model(&domain.OssPolicy{}).
        Where("id = ? AND school_id = ?", policyID, schoolID).
        Updates(map[string]any{
            "status":          status,
            "last_updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
        })
    if result.Error != nil {
        return nil, result.Error
    }
    if result.RowsAffected == 0 {
        return nil, gorm.ErrRecordNotFound
    }
    return s.GetByID(ctx, policyID, schoolID)
}

var _ repository.OssPolicyRepository = (*OssPolicyStore)(nil)

// OssAuditStore implements repository.OssAuditRepository using GORM.
type OssAuditStore struct {
    db *gorm.DB
}

// NewOssAuditStore creates a new OssAuditStore.
func NewOssAuditStore(db *gorm.DB) *OssAuditStore {
    return &OssAuditStore{db: db}
}

func (s *OssAuditStore) Create(ctx context.Context, log *domain.OssAuditLog) error {
    return s.db.WithContext(ctx).Create(log).Error
}

func (s *OssAuditStore) ListRecent(ctx context.Context, schoolID string, limit int) ([]domain.OssAuditLog, error) {
    if limit <= 0 {
        limit = 20
    }
    var logs []domain.OssAuditLog
    if err := s.db.WithContext(ctx).
        Where("school_id = ?", schoolID).
        Order("created_at DESC").
        Limit(limit).
        Find(&logs).Error; err != nil {
        return nil, err
    }
    return logs, nil
}

var _ repository.OssAuditRepository = (*OssAuditStore)(nil)
