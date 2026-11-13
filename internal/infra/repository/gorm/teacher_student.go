package gormrepo

import (
	"context"

	"github.com/google/uuid"

	"learn-go/internal/domain/roster"

	"gorm.io/gorm"
)

// TeacherStudentStore 管理教师与学生的关联。
type TeacherStudentStore struct {
	db *gorm.DB
}

func NewTeacherStudentStore(db *gorm.DB) *TeacherStudentStore {
	return &TeacherStudentStore{db: db}
}

func (s *TeacherStudentStore) BindTeachers(ctx context.Context, studentID string, teacherIDs []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("student_id = ?", studentID).Delete(&roster.TeacherStudentLink{}).Error; err != nil {
			return err
		}

		links := make([]roster.TeacherStudentLink, 0, len(teacherIDs))
		for _, teacherID := range teacherIDs {
			if teacherID == "" {
				continue
			}
			links = append(links, roster.TeacherStudentLink{
				ID:        uuid.NewString(),
				TeacherID: teacherID,
				StudentID: studentID,
			})
		}

		if len(links) > 0 {
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

var _ roster.TeacherStudentRepository = (*TeacherStudentStore)(nil)
