package gormrepo

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"learn-go/internal/domain"
)

type CourseTeacherStore struct {
	db *gorm.DB
}

func NewCourseTeacherStore(db *gorm.DB) *CourseTeacherStore {
	return &CourseTeacherStore{db: db}
}

func (s *CourseTeacherStore) Add(ctx context.Context, courseID string, teacherIDs []string) error {
	if len(teacherIDs) == 0 {
		return nil
	}
	var links []domain.CourseTeacher
	for _, tid := range teacherIDs {
		links = append(links, domain.CourseTeacher{
			ID:        uuid.NewString(),
			CourseID:  courseID,
			TeacherID: tid,
		})
	}
	return s.db.WithContext(ctx).Create(&links).Error
}

func (s *CourseTeacherStore) Remove(ctx context.Context, courseID string, teacherIDs []string) error {
	if len(teacherIDs) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Where("course_id = ? AND teacher_id IN ?", courseID, teacherIDs).
		Delete(&domain.CourseTeacher{}).Error
}

func (s *CourseTeacherStore) ListByCourseID(ctx context.Context, courseID string) ([]domain.CourseTeacher, error) {
	var links []domain.CourseTeacher
	err := s.db.WithContext(ctx).Where("course_id = ?", courseID).Find(&links).Error
	return links, err
}
