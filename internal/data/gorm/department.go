package gormrepo

import (
	"context"
	"gorm.io/gorm"
	schoolbiz "learn-go/internal/biz/school"
)

// DepartmentStore implements schoolbiz.DepartmentRepository using GORM.
type DepartmentStore struct {
	db *gorm.DB
}

func NewDepartmentStore(db *gorm.DB) *DepartmentStore {
	return &DepartmentStore{db: db}
}

func (s *DepartmentStore) Create(ctx context.Context, department *schoolbiz.Department) error {
	return s.db.WithContext(ctx).Create(department).Error
}

func (s *DepartmentStore) List(ctx context.Context, schoolID string) ([]schoolbiz.Department, error) {
	var departments []schoolbiz.Department
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

func (s *DepartmentStore) GetByID(ctx context.Context, id string) (*schoolbiz.Department, error) {
	var department schoolbiz.Department
	if err := s.db.WithContext(ctx).First(&department, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &department, nil
}

func (s *DepartmentStore) UpdateName(ctx context.Context, id, schoolID, name string) error {
	return s.db.WithContext(ctx).
		Model(&schoolbiz.Department{}).
		Where("id = ? AND school_id = ?", id, schoolID).
		Update("name", name).Error
}

func (s *DepartmentStore) Delete(ctx context.Context, id, schoolID string) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", id, schoolID).
		Delete(&schoolbiz.Department{}).Error
}

var _ schoolbiz.DepartmentRepository = (*DepartmentStore)(nil)
