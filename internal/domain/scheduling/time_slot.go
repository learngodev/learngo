package scheduling

import (
	"time"
)

// TimeSlot 定义一个上课节次。
type TimeSlot struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID  string    `gorm:"size:36;index" json:"school_id"`
	Name      string    `gorm:"size:64" json:"name"`      // e.g., "第1-2节"
	StartTime string    `gorm:"size:5" json:"start_time"` // 格式 HH:mm
	EndTime   string    `gorm:"size:5" json:"end_time"`   // 格式 HH:mm
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CourseSlot 是 TimeSlot 的别名，用于部分场景。
type CourseSlot struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID  string    `gorm:"size:36;index" json:"school_id"`
	Name      string    `gorm:"size:64" json:"name"`
	StartTime string    `gorm:"size:5" json:"start_time"`
	EndTime   string    `gorm:"size:5" json:"end_time"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (CourseSlot) TableName() string {
	return "time_slots"
}
