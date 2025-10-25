package gormrepo

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// DepartmentStore implements repository.DepartmentRepository using GORM.
type DepartmentStore struct {
	db *gorm.DB
}

func NewDepartmentStore(db *gorm.DB) *DepartmentStore {
	return &DepartmentStore{db: db}
}

func (s *DepartmentStore) Create(ctx context.Context, department *domain.Department) error {
	return s.db.WithContext(ctx).Create(department).Error
}

func (s *DepartmentStore) List(ctx context.Context, schoolID string) ([]domain.Department, error) {
	var departments []domain.Department
	if err := s.db.WithContext(ctx).Where("school_id = ?", schoolID).Order("created_at").Find(&departments).Error; err != nil {
		return nil, err
	}
	return departments, nil
}

func (s *DepartmentStore) GetByID(ctx context.Context, id string) (*domain.Department, error) {
	var department domain.Department
	if err := s.db.WithContext(ctx).First(&department, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &department, nil
}

func (s *DepartmentStore) UpdateName(ctx context.Context, id, schoolID, name string) error {
	return s.db.WithContext(ctx).
		Model(&domain.Department{}).
		Where("id = ? AND school_id = ?", id, schoolID).
		Update("name", name).Error
}

func (s *DepartmentStore) Delete(ctx context.Context, id, schoolID string) error {
	return s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", id, schoolID).
		Delete(&domain.Department{}).Error
}

var _ repository.DepartmentRepository = (*DepartmentStore)(nil)
