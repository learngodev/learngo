package roster

import (
	"time"
)

// Student 表示学生档案。
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

	// 虚拟字段（只读）
	Name string `gorm:"->;<-:false" json:"name"`
}
