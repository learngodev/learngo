package system

import "time"

// SystemBroadcast represents announcement metadata pushed to administrators.
type SystemBroadcast struct {
	ID             string `gorm:"primaryKey;size:64"`
	SchoolID       string `gorm:"size:36;index"`
	Title          string `gorm:"size:256"`
	MessagePreview string `gorm:"size:512"`
	Status         string `gorm:"size:32"`
	TargetLabel    string `gorm:"size:128"`
	ScheduleLabel  string `gorm:"size:128"`
	CreatedBy      string `gorm:"size:128"`
	Pinned         bool
	SortOrder      int `gorm:"index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
