package curriculum

import (
	"time"
)

// CourseChapter 表示教师编写的课后学习内容。
type CourseChapter struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	CourseID   string    `gorm:"size:36;index" json:"course_id"`
	TeacherID  string    `gorm:"size:36;index" json:"teacher_id"`
	Title      string    `gorm:"size:256" json:"title"`
	Content    string    `gorm:"type:text" json:"content"`
	OrderIndex int       `gorm:"default:0" json:"order_index"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CourseChapterAttachment 关联课程章节与附件文件。
type CourseChapterAttachment struct {
	ID        string `gorm:"primaryKey;size:36"`
	ChapterID string `gorm:"size:36;index"`
	FileID    string `gorm:"size:36;index"`
	CreatedAt time.Time
}
