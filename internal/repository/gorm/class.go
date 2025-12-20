package gormrepo

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// ClassStore implements repository.ClassRepository using GORM.
type ClassStore struct {
	db *gorm.DB
}

func NewClassStore(db *gorm.DB) *ClassStore {
	return &ClassStore{db: db}
}

func (s *ClassStore) Create(ctx context.Context, class *domain.Class) error {
	return s.db.WithContext(ctx).Create(class).Error
}

func (s *ClassStore) ListByDepartment(ctx context.Context, schoolID, departmentID string) ([]domain.Class, error) {
	var classes []domain.Class
	query := s.db.WithContext(ctx).Table("classes").
		Select("classes.*, "+
			"(SELECT count(*) FROM students WHERE students.class_id = classes.id) as student_count, "+
			"(SELECT count(DISTINCT teacher_id) FROM course_schedules WHERE course_schedules.class_id = classes.id) as teacher_count").
		Where("classes.school_id = ?", schoolID)

	if departmentID != "" {
		query = query.Where("classes.department_id = ?", departmentID)
	}
	if err := query.Order("classes.created_at").Scan(&classes).Error; err != nil {
		return nil, err
	}
	return classes, nil
}

func (s *ClassStore) GetByID(ctx context.Context, id string) (*domain.Class, error) {
	var class domain.Class
	if err := s.db.WithContext(ctx).First(&class, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &class, nil
}

func (s *ClassStore) ListByIDs(ctx context.Context, ids []string) ([]domain.Class, error) {
	if len(ids) == 0 {
		return []domain.Class{}, nil
	}
	var classes []domain.Class
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&classes).Error; err != nil {
		return nil, err
	}
	return classes, nil
}

func (s *ClassStore) UpdateName(ctx context.Context, id, schoolID, name string) error {
	return s.db.WithContext(ctx).
		Model(&domain.Class{}).
		Where("id = ? AND school_id = ?", id, schoolID).
		Update("name", name).Error
}

func (s *ClassStore) Delete(ctx context.Context, id, schoolID string) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", id, schoolID).
		Delete(&domain.Class{}).Error
}

var _ repository.ClassRepository = (*ClassStore)(nil)
