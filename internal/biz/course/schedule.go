package course

import "time"

// CourseSchedule defines a recurring class rule.
type CourseSchedule struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID    string    `gorm:"size:36;index" json:"school_id"`
	CourseID    string    `gorm:"size:36;index" json:"course_id"`
	ClassID     string    `gorm:"size:36;index" json:"class_id"`
	TeacherID   *string   `gorm:"size:36;index" json:"teacher_id"`
	SlotID      string    `gorm:"size:36;index" json:"slot_id"`
	ClassroomID *string   `gorm:"size:36;index" json:"classroom_id"`
	DayOfWeek   int       `gorm:"index" json:"day_of_week"`
	Location    string    `gorm:"size:128" json:"location"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CourseScheduleDetail extends CourseSchedule with related entity names.
type CourseScheduleDetail struct {
	CourseSchedule
	CourseName   string `json:"course_name"`
	ClassName    string `json:"class_name"`
	TeacherName  string `json:"teacher_name"`
	SlotName     string `json:"slot_name"`
	ClassroomLoc string `json:"classroom_location"`
}

// ScheduleStats aggregates schedule statistics.
type ScheduleStats struct {
	TotalRules              int64         `json:"total_rules"`
	TotalCourses            int64         `json:"total_courses"`
	ScheduledCoursesCount   int64         `json:"scheduled_courses_count"`
	UnscheduledCoursesCount int64         `json:"unscheduled_courses_count"`
	RulesByDay              map[int]int64 `json:"rules_by_day"`
}
