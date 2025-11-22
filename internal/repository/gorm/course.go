package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"

	"gorm.io/gorm"
)

// CourseStore implements repository.CourseRepository.
type CourseStore struct {
	db *gorm.DB
}

// NewCourseStore constructs CourseStore.
func NewCourseStore(db *gorm.DB) *CourseStore {
	return &CourseStore{db: db}
}

// ListByIDs returns course metadata for the provided identifiers.
func (s *CourseStore) ListByIDs(ctx context.Context, ids []string) ([]domain.Course, error) {
	if len(ids) == 0 {
		return []domain.Course{}, nil
	}

	var courses []domain.Course
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}

// CourseSessionStore implements repository.CourseSessionRepository.
type CourseSessionStore struct {
	db *gorm.DB
}

// NewCourseSessionStore constructs CourseSessionStore.
func NewCourseSessionStore(db *gorm.DB) *CourseSessionStore {
	return &CourseSessionStore{db: db}
}

// ListByClassBetween returns sessions for a class within the provided time window.
func (s *CourseSessionStore) ListByClassBetween(ctx context.Context, classID string, start, end time.Time) ([]domain.CourseSession, error) {
	query := s.db.WithContext(ctx).Where("class_id = ?", classID)
	if !start.IsZero() {
		query = query.Where("starts_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("starts_at < ?", end)
	}
	query = query.Order("starts_at ASC")

	var sessions []domain.CourseSession
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// ListByTeacherBetween returns sessions taught by a teacher within the time window.
func (s *CourseSessionStore) ListByTeacherBetween(ctx context.Context, teacherID string, start, end time.Time) ([]domain.CourseSession, error) {
	query := s.db.WithContext(ctx).Where("teacher_id = ?", teacherID)
	if !start.IsZero() {
		query = query.Where("starts_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("starts_at < ?", end)
	}
	query = query.Order("starts_at ASC")

	var sessions []domain.CourseSession
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}
