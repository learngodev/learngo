package gormrepo

import (
	"context"

	"learn-go/internal/domain/organization"

	"gorm.io/gorm"
)

// DepartmentStore 基于 GORM 实现 organization.DepartmentRepository。
type DepartmentStore struct {
	db *gorm.DB
}

func NewDepartmentStore(db *gorm.DB) *DepartmentStore {
	return &DepartmentStore{db: db}
}

func (s *DepartmentStore) Create(ctx context.Context, department *organization.Department) error {
	return s.db.WithContext(ctx).Create(department).Error
}

func (s *DepartmentStore) List(ctx context.Context, schoolID string) ([]organization.Department, error) {
	var departments []organization.Department
	err := s.db.WithContext(ctx).
		Table("departments").
		Select("departments.*, "+
			"(SELECT count(DISTINCT course_schedules.teacher_id) FROM course_schedules JOIN classes ON course_schedules.class_id = classes.id WHERE classes.department_id = departments.id AND course_schedules.teacher_id IS NOT NULL) as teacher_count, "+
			"(SELECT count(*) FROM students JOIN classes ON students.class_id = classes.id WHERE classes.department_id = departments.id) as student_count").
		Where("departments.school_id = ?", schoolID).
		Order("departments.created_at").
		Scan(&departments).Error

	if err != nil {
		return nil, err
	}
	return departments, nil
}

func (s *DepartmentStore) GetByID(ctx context.Context, id string) (*organization.Department, error) {
	var department organization.Department
	if err := s.db.WithContext(ctx).First(&department, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	return &department, nil
}

func (s *DepartmentStore) UpdateName(ctx context.Context, id, schoolID, name string) error {
	return s.db.WithContext(ctx).
		Model(&organization.Department{}).
		Where("id = ? AND school_id = ?", id, schoolID).
		Update("name", name).Error
}

func (s *DepartmentStore) Delete(ctx context.Context, id, schoolID string) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", id, schoolID).
		Delete(&organization.Department{}).Error
}

var _ organization.DepartmentRepository = (*DepartmentStore)(nil)
