package storage

import (
	"context"
)

// FileRepository 负责已上传文件记录的持久化。
type FileRepository interface {
	Create(ctx context.Context, file *File) error
	GetByID(ctx context.Context, id, schoolID string) (*File, error)
	GetByUploader(ctx context.Context, id, schoolID, uploaderID string) (*File, error)
	ListByIDs(ctx context.Context, ids []string) ([]File, error)
}

// OssCredentialRepository 负责 OSS 凭据的持久化。
type OssCredentialRepository interface {
	Create(ctx context.Context, credential *OssCredential) error
	List(ctx context.Context, schoolID string) ([]OssCredential, error)
	GetByID(ctx context.Context, credentialID, schoolID string) (*OssCredential, error)
	Update(ctx context.Context, credentialID, schoolID string, updates map[string]any) (*OssCredential, error)
	SetPrimary(ctx context.Context, credentialID, schoolID string) (*OssCredential, error)
	GetPrimary(ctx context.Context, schoolID string) (*OssCredential, error)
	Delete(ctx context.Context, credentialID, schoolID string) error
}

// OssPolicyRepository 负责 OSS 策略的持久化。
type OssPolicyRepository interface {
	Create(ctx context.Context, policy *OssPolicy) error
	List(ctx context.Context, schoolID string) ([]OssPolicy, error)
	GetByID(ctx context.Context, policyID, schoolID string) (*OssPolicy, error)
	UpdateStatus(ctx context.Context, policyID, schoolID string, status OssPolicyStatus) (*OssPolicy, error)
	Delete(ctx context.Context, policyID, schoolID string) error
}

// OssAuditRepository 记录 OSS 配置变更。
type OssAuditRepository interface {
	Create(ctx context.Context, log *OssAuditLog) error
	ListRecent(ctx context.Context, schoolID string, limit int) ([]OssAuditLog, error)
}
