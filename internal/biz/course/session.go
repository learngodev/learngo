package course

import "time"

// CourseSession is a scheduled lesson.
type CourseSession struct {
	ID          string  `gorm:"primaryKey;size:36"`
	CourseID    string  `gorm:"size:36;index"`
	ClassID     string  `gorm:"size:36;index"`
	TeacherID   *string `gorm:"size:36;index"`
	SlotID      string  `gorm:"size:36;index"`
	ClassroomID *string `gorm:"size:36;index"`
	StartsAt    time.Time
	EndsAt      time.Time
	Location    string `gorm:"size:128"`
	Source      string `gorm:"size:32"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
