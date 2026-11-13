package gormrepo

import (
	"context"

	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/internal/infra/repository"

	"gorm.io/gorm"
)

// TeacherStudentStore manages teacher-student associations.
type TeacherStudentStore struct {
	db *gorm.DB
}

func NewTeacherStudentStore(db *gorm.DB) *TeacherStudentStore {
	return &TeacherStudentStore{db: db}
}

func (s *TeacherStudentStore) BindTeachers(ctx context.Context, studentID string, teacherIDs []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("student_id = ?", studentID).Delete(&domain.TeacherStudentLink{}).Error; err != nil {
			return err
		}

		links := make([]domain.TeacherStudentLink, 0, len(teacherIDs))
		for _, teacherID := range teacherIDs {
			if teacherID == "" {
				continue
			}
			links = append(links, domain.TeacherStudentLink{
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

var _ repository.TeacherStudentRepository = (*TeacherStudentStore)(nil)
