package note

import (
	"time"

	"learn-go/internal/biz/shared"
)

// Note stores personal or shared notes.
type Note struct {
	ID         string      `gorm:"primaryKey;size:36"`
	SchoolID   string      `gorm:"size:36;index"`
	OwnerID    string      `gorm:"size:36;index"`
	OwnerRole  shared.Role `gorm:"size:16"`
	Title      string      `gorm:"size:256"`
	Content    string      `gorm:"type:text"`
	Visibility string      `gorm:"size:16"`
	Status     string      `gorm:"size:16"`
	DeletedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
