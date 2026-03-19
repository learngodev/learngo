package storage

import "time"

// OssPolicyStatus indicates policy enforcement state.
type OssPolicyStatus string

const (
	OssPolicyStatusEnabled  OssPolicyStatus = "enabled"
	OssPolicyStatusReadOnly OssPolicyStatus = "read_only"
	OssPolicyStatusDisabled OssPolicyStatus = "disabled"
)

// OssCredential stores object storage credential metadata.
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

// OssPolicy describes logical access restrictions for different scenarios.
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

// OssAuditLog records administrative changes made to OSS settings.
type OssAuditLog struct {
	ID           string    `gorm:"primaryKey;size:36"`
	SchoolID     string    `gorm:"size:36;index"`
	Action       string    `gorm:"size:128"`
	OperatorID   string    `gorm:"size:36"`
	OperatorName string    `gorm:"size:128"`
	Detail       string    `gorm:"size:512"`
	CreatedAt    time.Time `gorm:"index"`
}
