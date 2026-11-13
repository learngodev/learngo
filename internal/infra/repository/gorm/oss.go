package gormrepo

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/domain/storage"

	"gorm.io/gorm"
)

// OssCredentialStore 基于 GORM 实现 storage.OssCredentialRepository。
type OssCredentialStore struct {
	db *gorm.DB
}

// NewOssCredentialStore 创建 OssCredentialStore。
func NewOssCredentialStore(db *gorm.DB) *OssCredentialStore {
	return &OssCredentialStore{db: db}
}

func (s *OssCredentialStore) List(ctx context.Context, schoolID string) ([]storage.OssCredential, error) {
	var out []storage.OssCredential
	if err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("is_primary DESC, created_at DESC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *OssCredentialStore) Create(ctx context.Context, credential *storage.OssCredential) error {
	return s.db.WithContext(ctx).Create(credential).Error
}

func (s *OssCredentialStore) GetByID(ctx context.Context, credentialID, schoolID string) (*storage.OssCredential, error) {
	var credential storage.OssCredential
	if err := s.db.WithContext(ctx).
		First(&credential, "id = ? AND school_id = ?", credentialID, schoolID).Error; err != nil {
		return nil, notFound(err)
	}
	return &credential, nil
}

func (s *OssCredentialStore) Update(ctx context.Context, credentialID, schoolID string, updates map[string]any) (*storage.OssCredential, error) {
	if len(updates) == 0 {
		return s.GetByID(ctx, credentialID, schoolID)
	}
	result := s.db.WithContext(ctx).
		Model(&storage.OssCredential{}).
		Where("id = ? AND school_id = ?", credentialID, schoolID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	return s.GetByID(ctx, credentialID, schoolID)
}

func (s *OssCredentialStore) SetPrimary(ctx context.Context, credentialID, schoolID string) (*storage.OssCredential, error) {
	var updated *storage.OssCredential
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&storage.OssCredential{}).
			Where("school_id = ?", schoolID).
			Updates(map[string]any{"is_primary": false}).Error; err != nil {
			return err
		}
		result := tx.Model(&storage.OssCredential{}).
			Where("id = ? AND school_id = ?", credentialID, schoolID).
			Updates(map[string]any{"is_primary": true, "active": true})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		var cred storage.OssCredential
		if err := tx.First(&cred, "id = ? AND school_id = ?", credentialID, schoolID).Error; err != nil {
			return notFound(err)
		}
		updated = &cred
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *OssCredentialStore) GetPrimary(ctx context.Context, schoolID string) (*storage.OssCredential, error) {
	var credential storage.OssCredential
	if err := s.db.WithContext(ctx).
		Where("school_id = ? AND is_primary = ? AND active = ?", schoolID, true, true).
		First(&credential).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &credential, nil
}

func (s *OssCredentialStore) Delete(ctx context.Context, credentialID, schoolID string) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", credentialID, schoolID).
		Delete(&storage.OssCredential{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

var _ storage.OssCredentialRepository = (*OssCredentialStore)(nil)

// OssPolicyStore 基于 GORM 实现 storage.OssPolicyRepository。
type OssPolicyStore struct {
	db *gorm.DB
}

// NewOssPolicyStore 创建 OssPolicyStore。
func NewOssPolicyStore(db *gorm.DB) *OssPolicyStore {
	return &OssPolicyStore{db: db}
}

func (s *OssPolicyStore) List(ctx context.Context, schoolID string) ([]storage.OssPolicy, error) {
	var out []storage.OssPolicy
	if err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("created_at ASC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *OssPolicyStore) Create(ctx context.Context, policy *storage.OssPolicy) error {
	return s.db.WithContext(ctx).Create(policy).Error
}

func (s *OssPolicyStore) GetByID(ctx context.Context, policyID, schoolID string) (*storage.OssPolicy, error) {
	var policy storage.OssPolicy
	if err := s.db.WithContext(ctx).
		First(&policy, "id = ? AND school_id = ?", policyID, schoolID).Error; err != nil {
		return nil, notFound(err)
	}
	return &policy, nil
}

func (s *OssPolicyStore) UpdateStatus(ctx context.Context, policyID, schoolID string, status storage.OssPolicyStatus) (*storage.OssPolicy, error) {
	result := s.db.WithContext(ctx).
		Model(&storage.OssPolicy{}).
		Where("id = ? AND school_id = ?", policyID, schoolID).
		Updates(map[string]any{
			"status":          status,
			"last_updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	return s.GetByID(ctx, policyID, schoolID)
}

func (s *OssPolicyStore) Delete(ctx context.Context, policyID, schoolID string) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", policyID, schoolID).
		Delete(&storage.OssPolicy{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

var _ storage.OssPolicyRepository = (*OssPolicyStore)(nil)

// OssAuditStore 基于 GORM 实现 storage.OssAuditRepository。
type OssAuditStore struct {
	db *gorm.DB
}

// NewOssAuditStore 创建 OssAuditStore。
func NewOssAuditStore(db *gorm.DB) *OssAuditStore {
	return &OssAuditStore{db: db}
}

func (s *OssAuditStore) Create(ctx context.Context, log *storage.OssAuditLog) error {
	return s.db.WithContext(ctx).Create(log).Error
}

func (s *OssAuditStore) ListRecent(ctx context.Context, schoolID string, limit int) ([]storage.OssAuditLog, error) {
	if limit <= 0 {
		limit = 20
	}
	var logs []storage.OssAuditLog
	if err := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

var _ storage.OssAuditRepository = (*OssAuditStore)(nil)
