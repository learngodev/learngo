package course

import "time"

// TimeSlot defines a class period.
type TimeSlot struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID  string    `gorm:"size:36;index" json:"school_id"`
	Name      string    `gorm:"size:64" json:"name"`
	StartTime string    `gorm:"size:5" json:"start_time"`
	EndTime   string    `gorm:"size:5" json:"end_time"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CourseSlot is an alias for TimeSlot, used in some contexts.
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
