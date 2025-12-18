package gormrepo

import (
	"context"

	"learn-go/internal/domain"

	"gorm.io/gorm"
)

// CourseStudentStore implements repository.CourseStudentRepository.
type CourseStudentStore struct {
	db *gorm.DB
}

// NewCourseStudentStore constructs CourseStudentStore.
func NewCourseStudentStore(db *gorm.DB) *CourseStudentStore {
	return &CourseStudentStore{db: db}
}

// BatchCreate enrolls multiple students in a course.
func (s *CourseStudentStore) BatchCreate(ctx context.Context, enrollments []domain.CourseStudent) error {
	if len(enrollments) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Create(&enrollments).Error
}

// ListByCourseID retrieves all students enrolled in a course.
func (s *CourseStudentStore) ListByCourseID(ctx context.Context, courseID string) ([]domain.CourseStudent, error) {
	var enrollments []domain.CourseStudent
	if err := s.db.WithContext(ctx).Where("course_id = ?", courseID).Find(&enrollments).Error; err != nil {
		return nil, err
	}
	return enrollments, nil
}

// DeleteByCourseAndStudent removes students from a course.
func (s *CourseStudentStore) DeleteByCourseAndStudent(ctx context.Context, courseID string, studentIDs []string) error {
	if len(studentIDs) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Where("course_id = ? AND student_id IN ?", courseID, studentIDs).Delete(&domain.CourseStudent{}).Error
}
