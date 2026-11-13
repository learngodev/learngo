package curriculum

import (
	"time"

	"learn-go/internal/domain/roster"
)

// Course 表示由教师讲授的一门课程。
type Course struct {
	ID             string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID       string    `gorm:"size:36;index" json:"school_id"`
	Name           string    `gorm:"size:128" json:"name"`
	Description    string    `gorm:"size:512" json:"description"`
	ImageURL       string    `gorm:"size:256" json:"image_url"`
	InvitationCode string    `gorm:"size:32;index" json:"invitation_code"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// 关联关系
	Teachers []roster.Teacher `gorm:"many2many:course_teachers;" json:"teachers,omitempty"`
}

// CourseAssignmentInfo 表示补充了关联信息的课程分配数据。
type CourseAssignmentInfo struct {
	AssignmentID string `json:"assignment_id"`
	CourseID     string `json:"course_id"`
	CourseName   string `json:"course_name"`
	Description  string `json:"description"`
	ImageURL     string `json:"image_url"`
	TeacherID    string `json:"teacher_id"`
	TeacherName  string `json:"teacher_name"`
	ClassID      string `json:"class_id"`
	ClassName    string `json:"class_name"`
	StudentCount int64  `json:"student_count"`
}

// CourseStudent 关联学生与课程（选课）。
type CourseStudent struct {
	ID        string `gorm:"primaryKey;size:36"`
	CourseID  string `gorm:"size:36;index"`
	StudentID string `gorm:"size:36;index"`
	CreatedAt time.Time
}

// CourseTeacher 关联教师与课程。
type CourseTeacher struct {
	ID        string `gorm:"primaryKey;size:36"`
	CourseID  string `gorm:"size:36;index"`
	TeacherID string `gorm:"size:36;index"`
	CreatedAt time.Time
}
