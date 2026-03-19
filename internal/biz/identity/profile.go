package identity

import (
	"time"
)

// Teacher profile.
type Teacher struct {
	ID           string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID     string    `gorm:"size:36;index" json:"school_id"`
	AccountID    string    `gorm:"size:36;uniqueIndex" json:"account_id"`
	Number       string    `gorm:"size:64;uniqueIndex" json:"number"`
	DepartmentID *string   `gorm:"size:36;index" json:"department_id"`
	Email        *string   `gorm:"size:128" json:"email"`
	Phone        string    `gorm:"size:32" json:"phone"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Student profile.
type Student struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID  string    `gorm:"size:36;index" json:"school_id"`
	AccountID string    `gorm:"size:36;uniqueIndex" json:"account_id"`
	Number    string    `gorm:"size:64;uniqueIndex" json:"number"`
	ClassID   *string   `gorm:"size:36;index" json:"class_id"`
	Email     *string   `gorm:"size:128" json:"email"`
	Phone     string    `gorm:"size:32" json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Name string `gorm:"->;<-:false" json:"name"`
}

// TeacherStudentLink binds teacher(s) to student.
type TeacherStudentLink struct {
	ID        string `gorm:"primaryKey;size:36"`
	TeacherID string `gorm:"size:36;index"`
	StudentID string `gorm:"size:36;index"`
	CreatedAt time.Time
}
