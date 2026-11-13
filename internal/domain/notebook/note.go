package notebook

import (
	"time"

	"learn-go/internal/domain/identity"
)

// Note 存储个人或共享笔记。
type Note struct {
	ID         string        `gorm:"primaryKey;size:36"`
	SchoolID   string        `gorm:"size:36;index"`
	OwnerID    string        `gorm:"size:36;index"`
	OwnerRole  identity.Role `gorm:"size:16"`
	Title      string        `gorm:"size:256"`
	Content    string        `gorm:"type:text"`
	Visibility string        `gorm:"size:16"` // private（私有）、class（班级）、school（全校）
	Status     string        `gorm:"size:16"` // draft（草稿）、published（已发布）
	DeletedAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NoteComment 表示笔记下的协作评论。
type NoteComment struct {
	ID         string        `gorm:"primaryKey;size:36"`
	NoteID     string        `gorm:"size:36;index"`
	AuthorID   string        `gorm:"size:36;index"`
	AuthorRole identity.Role `gorm:"size:16"`
	Content    string        `gorm:"type:text"`
	CreatedAt  time.Time
}
