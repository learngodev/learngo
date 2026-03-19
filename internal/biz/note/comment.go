package note

import (
	"time"

	"learn-go/internal/biz/shared"
)

// NoteComment for collaborative feedback.
type NoteComment struct {
	ID         string      `gorm:"primaryKey;size:36"`
	NoteID     string      `gorm:"size:36;index"`
	AuthorID   string      `gorm:"size:36;index"`
	AuthorRole shared.Role `gorm:"size:16"`
	Content    string      `gorm:"type:text"`
	CreatedAt  time.Time
}
