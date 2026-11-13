package storage

import (
	"time"
)

// OssCredential 存储供管理员配置的对象存储凭据信息。
type OssCredential struct {
	ID                   string `gorm:"primaryKey;size:36"`
	SchoolID             string `gorm:"size:36;index"`
	Name                 string `gorm:"size:128"`
	Endpoint             string `gorm:"size:128"`
	InternalEndpoint     string `gorm:"size:128"`
	Region               string `gorm:"size:64"`
	Bucket               string `gorm:"size:128"`
	AccessKeyID          string `gorm:"size:128"`
	AccessKeySecret      string `gorm:"size:128"`
	AccessKeyDisplay     string `gorm:"size:128"`
	DirectoryPrefix      string `gorm:"size:128"`
	AllowPublicRead      bool   `gorm:"default:false"`
	AllowMultipartUpload bool   `gorm:"default:false"`
	UseRelayUpload       bool   `gorm:"default:false"`
	IsPrimary            bool   `gorm:"index"`
	Active               bool   `gorm:"index"`
	LastRotatedAt        *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// OssPolicyStatus 表示策略的执行状态。
type OssPolicyStatus string

const (
	OssPolicyStatusEnabled  OssPolicyStatus = "enabled"
	OssPolicyStatusReadOnly OssPolicyStatus = "read_only"
	OssPolicyStatusDisabled OssPolicyStatus = "disabled"
)

// OssPolicy 描述不同场景下的逻辑访问限制。
type OssPolicy struct {
	ID            string          `gorm:"primaryKey;size:36"`
	SchoolID      string          `gorm:"size:36;index"`
	Name          string          `gorm:"size:128"`
	Description   string          `gorm:"size:512"`
	AppliesTo     string          `gorm:"size:128"`
	Status        OssPolicyStatus `gorm:"size:32;index"`
	LastUpdatedAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OssAuditLog 记录管理员对 OSS 配置的变更。
type OssAuditLog struct {
	ID           string    `gorm:"primaryKey;size:36"`
	SchoolID     string    `gorm:"size:36;index"`
	Action       string    `gorm:"size:128"`
	OperatorID   string    `gorm:"size:36"`
	OperatorName string    `gorm:"size:128"`
	Detail       string    `gorm:"size:512"`
	CreatedAt    time.Time `gorm:"index"`
}
