package roster

import (
	"time"
)

// TeacherStudentLink 将教师与学生绑定。
type TeacherStudentLink struct {
	ID        string `gorm:"primaryKey;size:36"`
	TeacherID string `gorm:"size:36;index"`
	StudentID string `gorm:"size:36;index"`
	CreatedAt time.Time
}
