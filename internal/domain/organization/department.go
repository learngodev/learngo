package organization

import (
	"time"
)

// Department 按学科或年级对班级进行分组。
type Department struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID  string    `gorm:"size:36;index" json:"school_id"`
	Name      string    `gorm:"size:128" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 用于计数的虚拟字段（只读）
	TeacherCount int64 `gorm:"->;<-:false" json:"teacher_count"`
	StudentCount int64 `gorm:"->;<-:false" json:"student_count"`
}
