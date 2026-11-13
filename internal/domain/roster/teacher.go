package roster

import (
	"time"
)

// Teacher 表示教师档案。
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
