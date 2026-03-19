package gormrepo

import (
	"context"
	"gorm.io/gorm"
	coursebiz "learn-go/internal/biz/course"
)

type CourseScheduleStore struct {
	db *gorm.DB
}

func NewCourseScheduleStore(db *gorm.DB) *CourseScheduleStore {
	return &CourseScheduleStore{db: db}
}

func (s *CourseScheduleStore) Create(ctx context.Context, schedule *coursebiz.CourseSchedule) error {
	return s.db.WithContext(ctx).Create(schedule).Error
}

func (s *CourseScheduleStore) Update(ctx context.Context, schedule *coursebiz.CourseSchedule) error {
	return s.db.WithContext(ctx).Save(schedule).Error
}

func (s *CourseScheduleStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&coursebiz.CourseSchedule{}, "id = ?", id).Error
}

func (s *CourseScheduleStore) ListByCourse(ctx context.Context, courseID string) ([]coursebiz.CourseSchedule, error) {
	var schedules []coursebiz.CourseSchedule
	err := s.db.WithContext(ctx).Where("course_id = ?", courseID).Find(&schedules).Error
	return schedules, err
}

func (s *CourseScheduleStore) ListBySchool(ctx context.Context, schoolID string) ([]coursebiz.CourseSchedule, error) {
	var schedules []coursebiz.CourseSchedule
	err := s.db.WithContext(ctx).Where("school_id = ?", schoolID).Find(&schedules).Error
	return schedules, err
}

func (s *CourseScheduleStore) ListByTeacherID(ctx context.Context, teacherID string) ([]coursebiz.CourseSchedule, error) {
	var schedules []coursebiz.CourseSchedule
	err := s.db.WithContext(ctx).Where("teacher_id = ?", teacherID).Find(&schedules).Error
	return schedules, err
}

func (s *CourseScheduleStore) ListDetailsBySchool(ctx context.Context, schoolID string, courseID string) ([]coursebiz.CourseScheduleDetail, error) {
	var details []coursebiz.CourseScheduleDetail
	query := s.db.WithContext(ctx).
		Table("course_schedules").
		Select("course_schedules.*, courses.name as course_name, classes.name as class_name, accounts.display_name as teacher_name, time_slots.name as slot_name, classrooms.location as classroom_loc").
		Joins("LEFT JOIN courses ON course_schedules.course_id = courses.id").
		Joins("LEFT JOIN classes ON course_schedules.class_id = classes.id").
		Joins("LEFT JOIN teachers ON course_schedules.teacher_id = teachers.id").
		Joins("LEFT JOIN accounts ON teachers.account_id = accounts.id").
		Joins("LEFT JOIN time_slots ON course_schedules.slot_id = time_slots.id").
		Joins("LEFT JOIN classrooms ON course_schedules.classroom_id = classrooms.id").
		Where("course_schedules.school_id = ?", schoolID)

	if courseID != "" {
		query = query.Where("course_schedules.course_id = ?", courseID)
	}

	err := query.Scan(&details).Error
	return details, err
}

func (s *CourseScheduleStore) ListByClassroom(ctx context.Context, classroomID string) ([]coursebiz.CourseSchedule, error) {
	var schedules []coursebiz.CourseSchedule
	err := s.db.WithContext(ctx).Where("classroom_id = ?", classroomID).Find(&schedules).Error
	return schedules, err
}

func (s *CourseScheduleStore) ListByTeacher(ctx context.Context, teacherID string) ([]coursebiz.CourseSchedule, error) {
	var schedules []coursebiz.CourseSchedule
	err := s.db.WithContext(ctx).Where("teacher_id = ?", teacherID).Find(&schedules).Error
	return schedules, err
}

func (s *CourseScheduleStore) ListByClass(ctx context.Context, classID string) ([]coursebiz.CourseSchedule, error) {
	var schedules []coursebiz.CourseSchedule
	err := s.db.WithContext(ctx).Where("class_id = ?", classID).Find(&schedules).Error
	return schedules, err
}

func (s *CourseScheduleStore) GetStats(ctx context.Context, schoolID string) (*coursebiz.ScheduleStats, error) {
	var stats coursebiz.ScheduleStats

	// Total rules
	if err := s.db.WithContext(ctx).Model(&coursebiz.CourseSchedule{}).Where("school_id = ?", schoolID).Count(&stats.TotalRules).Error; err != nil {
		return nil, err
	}

	// Scheduled courses count (distinct course_id)
	if err := s.db.WithContext(ctx).Model(&coursebiz.CourseSchedule{}).Where("school_id = ?", schoolID).Distinct("course_id").Count(&stats.ScheduledCoursesCount).Error; err != nil {
		return nil, err
	}

	// Rules by day
	rows, err := s.db.WithContext(ctx).Model(&coursebiz.CourseSchedule{}).
		Select("day_of_week, count(*) as count").
		Where("school_id = ?", schoolID).
		Group("day_of_week").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.RulesByDay = make(map[int]int64)
	for rows.Next() {
		var day int
		var count int64
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		stats.RulesByDay[day] = count
	}

	return &stats, nil
}
