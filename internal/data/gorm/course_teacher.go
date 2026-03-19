package gormrepo

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	coursebiz "learn-go/internal/biz/course"
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
	var links []coursebiz.CourseTeacher
	for _, tid := range teacherIDs {
		links = append(links, coursebiz.CourseTeacher{
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
		Delete(&coursebiz.CourseTeacher{}).Error
}

func (s *CourseTeacherStore) ListByCourseID(ctx context.Context, courseID string) ([]coursebiz.CourseTeacher, error) {
	var links []coursebiz.CourseTeacher
	err := s.db.WithContext(ctx).Where("course_id = ?", courseID).Find(&links).Error
	return links, err
}

// ListCourseIDsByTeacher returns course IDs bound to the teacher.
func (s *CourseTeacherStore) ListCourseIDsByTeacher(ctx context.Context, teacherID string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).
		Model(&coursebiz.CourseTeacher{}).
		Where("teacher_id = ?", teacherID).
		Pluck("course_id", &ids).Error
	return ids, err
}
