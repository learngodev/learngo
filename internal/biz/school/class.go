package school

import "time"

// Class is a teaching class under a department.
type Class struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID     string    `gorm:"size:36;index" json:"school_id"`
	DepartmentID string    `gorm:"size:36;index" json:"department_id"`
	Name         string    `gorm:"size:128" json:"name"`
	HomeroomID   *string   `gorm:"size:36" json:"homeroom_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	StudentCount int64     `gorm:"->;<-:false" json:"student_count"`
	TeacherCount int64     `gorm:"->;<-:false" json:"teacher_count"`
}
