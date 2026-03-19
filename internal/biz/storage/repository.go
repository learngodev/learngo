package storage

import "context"

// OssCredentialRepository manages OSS credential persistence.
type OssCredentialRepository interface {
	Create(ctx context.Context, credential *OssCredential) error
	List(ctx context.Context, schoolID string) ([]OssCredential, error)
	GetByID(ctx context.Context, credentialID, schoolID string) (*OssCredential, error)
	Update(ctx context.Context, credentialID, schoolID string, updates map[string]any) (*OssCredential, error)
	SetPrimary(ctx context.Context, credentialID, schoolID string) (*OssCredential, error)
	GetPrimary(ctx context.Context, schoolID string) (*OssCredential, error)
	Delete(ctx context.Context, credentialID, schoolID string) error
}

// OssPolicyRepository manages OSS policy persistence.
type OssPolicyRepository interface {
	Create(ctx context.Context, policy *OssPolicy) error
	List(ctx context.Context, schoolID string) ([]OssPolicy, error)
	GetByID(ctx context.Context, policyID, schoolID string) (*OssPolicy, error)
	UpdateStatus(ctx context.Context, policyID, schoolID string, status OssPolicyStatus) (*OssPolicy, error)
	Delete(ctx context.Context, policyID, schoolID string) error
}

// OssAuditRepository records OSS configuration changes.
type OssAuditRepository interface {
	Create(ctx context.Context, log *OssAuditLog) error
	ListRecent(ctx context.Context, schoolID string, limit int) ([]OssAuditLog, error)
}
