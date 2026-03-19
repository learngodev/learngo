package system

import "time"

// SystemAuditLog captures administrative activity around system settings.
type SystemAuditLog struct {
	ID        string    `gorm:"primaryKey;size:36"`
	SchoolID  string    `gorm:"size:36;index"`
	Category  string    `gorm:"size:64"`
	Action    string    `gorm:"size:128"`
	Operator  string    `gorm:"size:128"`
	Detail    string    `gorm:"size:512"`
	TimeLabel string    `gorm:"size:64"`
	CreatedAt time.Time `gorm:"index"`
}
