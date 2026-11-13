package gormrepo

import (
	"context"

	"gorm.io/gorm"

	"learn-go/internal/domain/organization"
)

// ClassStore 基于 GORM 实现 organization.ClassRepository。
type ClassStore struct {
	db *gorm.DB
}

func NewClassStore(db *gorm.DB) *ClassStore {
	return &ClassStore{db: db}
}

func (s *ClassStore) Create(ctx context.Context, class *organization.Class) error {
	return s.db.WithContext(ctx).Create(class).Error
}

func (s *ClassStore) ListByDepartment(ctx context.Context, schoolID, departmentID string) ([]organization.Class, error) {
	var classes []organization.Class
	query := s.db.WithContext(ctx).Table("classes").
		Select("classes.*, "+
			"(SELECT count(*) FROM students WHERE students.class_id = classes.id) as student_count, "+
			"(SELECT count(DISTINCT teacher_id) FROM course_schedules WHERE course_schedules.class_id = classes.id AND course_schedules.teacher_id IS NOT NULL) as teacher_count").
		Where("classes.school_id = ?", schoolID)

	if departmentID != "" {
		query = query.Where("classes.department_id = ?", departmentID)
	}
	if err := query.Order("classes.created_at").Scan(&classes).Error; err != nil {
		return nil, err
	}
	return classes, nil
}

func (s *ClassStore) GetByID(ctx context.Context, id string) (*organization.Class, error) {
	var class organization.Class
	if err := s.db.WithContext(ctx).First(&class, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	return &class, nil
}

func (s *ClassStore) ListByIDs(ctx context.Context, ids []string) ([]organization.Class, error) {
	if len(ids) == 0 {
		return []organization.Class{}, nil
	}
	var classes []organization.Class
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&classes).Error; err != nil {
		return nil, err
	}
	return classes, nil
}

func (s *ClassStore) UpdateName(ctx context.Context, id, schoolID, name string) error {
	return s.db.WithContext(ctx).
		Model(&organization.Class{}).
		Where("id = ? AND school_id = ?", id, schoolID).
		Update("name", name).Error
}

func (s *ClassStore) Delete(ctx context.Context, id, schoolID string) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", id, schoolID).
		Delete(&organization.Class{}).Error
}

var _ organization.ClassRepository = (*ClassStore)(nil)
