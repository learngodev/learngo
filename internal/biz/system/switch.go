package system

import "time"

// SystemSwitch persists administrative feature toggles.
type SystemSwitch struct {
	ID               string `gorm:"primaryKey;size:64"`
	SchoolID         string `gorm:"size:36;index"`
	Title            string `gorm:"size:256"`
	Description      string `gorm:"size:512"`
	Enabled          bool
	LastUpdatedLabel string `gorm:"size:128"`
	Responsible      string `gorm:"size:128"`
	IconName         string `gorm:"size:64"`
	Tags             string `gorm:"size:256"`
	Environment      string `gorm:"size:64"`
	SortOrder        int    `gorm:"index"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
