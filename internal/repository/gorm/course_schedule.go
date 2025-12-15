package gormrepo

import (
	"context"

	"learn-go/internal/domain"

	"gorm.io/gorm"
)

type CourseScheduleStore struct {
	db *gorm.DB
}

func NewCourseScheduleStore(db *gorm.DB) *CourseScheduleStore {
	return &CourseScheduleStore{db: db}
}

func (s *CourseScheduleStore) Create(ctx context.Context, schedule *domain.CourseSchedule) error {
	return s.db.WithContext(ctx).Create(schedule).Error
}

func (s *CourseScheduleStore) Update(ctx context.Context, schedule *domain.CourseSchedule) error {
	return s.db.WithContext(ctx).Save(schedule).Error
}

func (s *CourseScheduleStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&domain.CourseSchedule{}, "id = ?", id).Error
}

func (s *CourseScheduleStore) ListByCourse(ctx context.Context, courseID string) ([]domain.CourseSchedule, error) {
	var schedules []domain.CourseSchedule
	err := s.db.WithContext(ctx).Where("course_id = ?", courseID).Find(&schedules).Error
	return schedules, err
}

func (s *CourseScheduleStore) ListBySchool(ctx context.Context, schoolID string) ([]domain.CourseSchedule, error) {
	var schedules []domain.CourseSchedule
	err := s.db.WithContext(ctx).Where("school_id = ?", schoolID).Find(&schedules).Error
	return schedules, err
}
