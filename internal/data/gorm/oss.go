package gormrepo

import (
	"context"
	"gorm.io/gorm"
	sharedbiz "learn-go/internal/biz/shared"
	storagebiz "learn-go/internal/biz/storage"
)

// OssCredentialStore implements storagebiz.OssCredentialRepository using GORM.
type OssCredentialStore struct {
	db *gorm.DB
}

// NewOssCredentialStore creates a new OssCredentialStore.
func NewOssCredentialStore(db *gorm.DB) *OssCredentialStore {
	return &OssCredentialStore{db: db}
}

func (s *OssCredentialStore) List(ctx context.Context, schoolID string) ([]storagebiz.OssCredential, error) {
	var out []storagebiz.OssCredential
	if err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("is_primary DESC, created_at DESC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *OssCredentialStore) Create(ctx context.Context, credential *storagebiz.OssCredential) error {
	return s.db.WithContext(ctx).Create(credential).Error
}

func (s *OssCredentialStore) GetByID(ctx context.Context, credentialID, schoolID string) (*storagebiz.OssCredential, error) {
	var credential storagebiz.OssCredential
	if err := s.db.WithContext(ctx).
		First(&credential, "id = ? AND school_id = ?", credentialID, schoolID).Error; err != nil {
		return nil, err
	}
	return &credential, nil
}

func (s *OssCredentialStore) Update(ctx context.Context, credentialID, schoolID string, updates map[string]any) (*storagebiz.OssCredential, error) {
	if len(updates) == 0 {
		return s.GetByID(ctx, credentialID, schoolID)
	}
	result := s.db.WithContext(ctx).
		Model(&storagebiz.OssCredential{}).
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

func (s *OssCredentialStore) SetPrimary(ctx context.Context, credentialID, schoolID string) (*storagebiz.OssCredential, error) {
	var updated *storagebiz.OssCredential
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&storagebiz.OssCredential{}).
			Where("school_id = ?", schoolID).
			Updates(map[string]any{"is_primary": false}).Error; err != nil {
			return err
		}
		result := tx.Model(&storagebiz.OssCredential{}).
			Where("id = ? AND school_id = ?", credentialID, schoolID).
			Updates(map[string]any{"is_primary": true, "active": true})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		var cred storagebiz.OssCredential
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

func (s *OssCredentialStore) GetPrimary(ctx context.Context, schoolID string) (*storagebiz.OssCredential, error) {
	var credential storagebiz.OssCredential
	if err := s.db.WithContext(ctx).
		Where("school_id = ? AND is_primary = ? AND active = ?", schoolID, true, true).
		First(&credential).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharedbiz.ErrNotFound
		}
		return nil, err
	}
	return &credential, nil
}

func (s *OssCredentialStore) Delete(ctx context.Context, credentialID, schoolID string) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", credentialID, schoolID).
		Delete(&storagebiz.OssCredential{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

var _ storagebiz.OssCredentialRepository = (*OssCredentialStore)(nil)

// OssPolicyStore implements storagebiz.OssPolicyRepository using GORM.
type OssPolicyStore struct {
	db *gorm.DB
}

// NewOssPolicyStore creates a new OssPolicyStore.
func NewOssPolicyStore(db *gorm.DB) *OssPolicyStore {
	return &OssPolicyStore{db: db}
}

func (s *OssPolicyStore) List(ctx context.Context, schoolID string) ([]storagebiz.OssPolicy, error) {
	var out []storagebiz.OssPolicy
	if err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("created_at ASC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *OssPolicyStore) Create(ctx context.Context, policy *storagebiz.OssPolicy) error {
	return s.db.WithContext(ctx).Create(policy).Error
}

func (s *OssPolicyStore) GetByID(ctx context.Context, policyID, schoolID string) (*storagebiz.OssPolicy, error) {
	var policy storagebiz.OssPolicy
	if err := s.db.WithContext(ctx).
		First(&policy, "id = ? AND school_id = ?", policyID, schoolID).Error; err != nil {
		return nil, err
	}
	return &policy, nil
}

func (s *OssPolicyStore) UpdateStatus(ctx context.Context, policyID, schoolID string, status storagebiz.OssPolicyStatus) (*storagebiz.OssPolicy, error) {
	result := s.db.WithContext(ctx).
		Model(&storagebiz.OssPolicy{}).
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

func (s *OssPolicyStore) Delete(ctx context.Context, policyID, schoolID string) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", policyID, schoolID).
		Delete(&storagebiz.OssPolicy{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

var _ storagebiz.OssPolicyRepository = (*OssPolicyStore)(nil)

// OssAuditStore implements storagebiz.OssAuditRepository using GORM.
type OssAuditStore struct {
	db *gorm.DB
}

// NewOssAuditStore creates a new OssAuditStore.
func NewOssAuditStore(db *gorm.DB) *OssAuditStore {
	return &OssAuditStore{db: db}
}

func (s *OssAuditStore) Create(ctx context.Context, log *storagebiz.OssAuditLog) error {
	return s.db.WithContext(ctx).Create(log).Error
}

func (s *OssAuditStore) ListRecent(ctx context.Context, schoolID string, limit int) ([]storagebiz.OssAuditLog, error) {
	if limit <= 0 {
		limit = 20
	}
	var logs []storagebiz.OssAuditLog
	if err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

var _ storagebiz.OssAuditRepository = (*OssAuditStore)(nil)
