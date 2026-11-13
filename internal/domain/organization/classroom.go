package organization

import (
	"time"
)

// Classroom 表示用于上课的实体教室。
type Classroom struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID  string    `gorm:"size:36;index" json:"school_id"`
	Location  string    `gorm:"size:128" json:"location"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
