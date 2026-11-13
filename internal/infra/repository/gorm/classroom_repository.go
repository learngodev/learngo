package gormrepo

import (
	"context"
	"errors"

	"learn-go/internal/domain"
	"learn-go/internal/domain/organization"

	"gorm.io/gorm"
)

type classroomRepository struct {
	db *gorm.DB
}

func NewClassroomRepository(db *gorm.DB) organization.ClassroomRepository {
	return &classroomRepository{db: db}
}

func (r *classroomRepository) Create(ctx context.Context, classroom *organization.Classroom) error {
	return r.db.WithContext(ctx).Create(classroom).Error
}

func (r *classroomRepository) GetByID(ctx context.Context, id string) (*organization.Classroom, error) {
	var classroom organization.Classroom
	if err := r.db.WithContext(ctx).First(&classroom, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &classroom, nil
}

func (r *classroomRepository) List(ctx context.Context, schoolID string, page, size int) ([]organization.Classroom, int64, error) {
	var classrooms []organization.Classroom
	var total int64

	db := r.db.WithContext(ctx).Model(&organization.Classroom{}).Where("school_id = ?", schoolID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if size > 0 {
		db = db.Offset((page - 1) * size).Limit(size)
	}

	if err := db.Find(&classrooms).Error; err != nil {
		return nil, 0, err
	}

	return classrooms, total, nil
}

func (r *classroomRepository) Update(ctx context.Context, classroom *organization.Classroom) error {
	return r.db.WithContext(ctx).Save(classroom).Error
}

func (r *classroomRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&organization.Classroom{}, "id = ?", id).Error
}
