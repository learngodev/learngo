package system

import "time"

// SystemParameter stores runtime platform parameters configurable by admins.
type SystemParameter struct {
	ID               string `gorm:"primaryKey;size:64"`
	SchoolID         string `gorm:"size:36;index"`
	Key              string `gorm:"size:128"`
	Value            string `gorm:"size:512"`
	Scope            string `gorm:"size:128"`
	Description      string `gorm:"size:512"`
	LastUpdatedLabel string `gorm:"size:128"`
	Locked           bool
	SortOrder        int `gorm:"index"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
