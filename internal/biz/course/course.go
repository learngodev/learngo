package course

import (
	"time"

	"learn-go/internal/biz/identity"
)

// Course represents a subject taught by a teacher.
type Course struct {
	ID             string             `gorm:"primaryKey;size:36" json:"id"`
	SchoolID       string             `gorm:"size:36;index" json:"school_id"`
	Name           string             `gorm:"size:128" json:"name"`
	Description    string             `gorm:"size:512" json:"description"`
	ImageURL       string             `gorm:"size:256" json:"image_url"`
	InvitationCode string             `gorm:"size:32;index" json:"invitation_code"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	Teachers       []identity.Teacher `gorm:"many2many:course_teachers;" json:"teachers,omitempty"`
}

// CourseAssignmentInfo represents enriched course assignment data.
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

// CourseStudent links a student to a course.
type CourseStudent struct {
	ID        string `gorm:"primaryKey;size:36"`
	CourseID  string `gorm:"size:36;index"`
	StudentID string `gorm:"size:36;index"`
	CreatedAt time.Time
}

// CourseTeacher links a teacher to a course.
type CourseTeacher struct {
	ID        string `gorm:"primaryKey;size:36"`
	CourseID  string `gorm:"size:36;index"`
	TeacherID string `gorm:"size:36;index"`
	CreatedAt time.Time
}
