package storage

import "time"

// File represents an uploaded file in OSS.
type File struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID   string    `gorm:"size:36;index" json:"school_id"`
	UploaderID string    `gorm:"size:36;index" json:"uploader_id"`
	Name       string    `gorm:"size:256" json:"name"`
	Key        string    `gorm:"size:256" json:"key"`
	URL        string    `gorm:"size:512" json:"url"`
	Type       string    `gorm:"size:255" json:"type"`
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
}
