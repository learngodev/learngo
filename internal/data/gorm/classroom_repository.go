package gormrepo

import (
	"context"
	"errors"
	"gorm.io/gorm"
	coursebiz "learn-go/internal/biz/course"
	sharedbiz "learn-go/internal/biz/shared"
)

type classroomRepository struct {
	db *gorm.DB
}

func NewClassroomRepository(db *gorm.DB) coursebiz.ClassroomRepository {
	return &classroomRepository{db: db}
}

func (r *classroomRepository) Create(ctx context.Context, classroom *coursebiz.Classroom) error {
	return r.db.WithContext(ctx).Create(classroom).Error
}

func (r *classroomRepository) GetByID(ctx context.Context, id string) (*coursebiz.Classroom, error) {
	var classroom coursebiz.Classroom
	if err := r.db.WithContext(ctx).First(&classroom, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sharedbiz.ErrNotFound
		}
		return nil, err
	}
	return &classroom, nil
}

func (r *classroomRepository) List(ctx context.Context, schoolID string, page, size int) ([]coursebiz.Classroom, int64, error) {
	var classrooms []coursebiz.Classroom
	var total int64

	db := r.db.WithContext(ctx).Model(&coursebiz.Classroom{}).Where("school_id = ?", schoolID)

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

func (r *classroomRepository) Update(ctx context.Context, classroom *coursebiz.Classroom) error {
	return r.db.WithContext(ctx).Save(classroom).Error
}

func (r *classroomRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&coursebiz.Classroom{}, "id = ?", id).Error
}
