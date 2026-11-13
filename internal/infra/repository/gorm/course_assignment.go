package gormrepo

import (
	"context"

	"learn-go/internal/domain/curriculum"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CourseStudentStore 实现 curriculum.CourseStudentRepository。
type CourseStudentStore struct {
	db *gorm.DB
}

// NewCourseStudentStore 构造 CourseStudentStore。
func NewCourseStudentStore(db *gorm.DB) *CourseStudentStore {
	return &CourseStudentStore{db: db}
}

// BatchCreate 将多名学生批量加入课程。
func (s *CourseStudentStore) BatchCreate(ctx context.Context, enrollments []curriculum.CourseStudent) error {
	if len(enrollments) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "course_id"}, {Name: "student_id"}},
			DoNothing: true,
		}).
		Create(&enrollments).Error
}

// ListByCourseID 查询某课程下的所有选课学生。
func (s *CourseStudentStore) ListByCourseID(ctx context.Context, courseID string) ([]curriculum.CourseStudent, error) {
	var enrollments []curriculum.CourseStudent
	if err := s.db.WithContext(ctx).Where("course_id = ?", courseID).Find(&enrollments).Error; err != nil {
		return nil, err
	}
	return enrollments, nil
}

// DeleteByCourseAndStudent 将学生移出课程。
func (s *CourseStudentStore) DeleteByCourseAndStudent(ctx context.Context, courseID string, studentIDs []string) error {
	if len(studentIDs) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Where("course_id = ? AND student_id IN ?", courseID, studentIDs).Delete(&curriculum.CourseStudent{}).Error
}

var _ curriculum.CourseStudentRepository = (*CourseStudentStore)(nil)
