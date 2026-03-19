package gormrepo

import (
	"context"
	"errors"
	"gorm.io/gorm"
	coursebiz "learn-go/internal/biz/course"
	sharedbiz "learn-go/internal/biz/shared"
)

type timeSlotRepository struct {
	db *gorm.DB
}

func NewTimeSlotRepository(db *gorm.DB) coursebiz.TimeSlotRepository {
	return &timeSlotRepository{db: db}
}

func (r *timeSlotRepository) Create(ctx context.Context, slot *coursebiz.TimeSlot) error {
	return r.db.WithContext(ctx).Create(slot).Error
}

func (r *timeSlotRepository) Update(ctx context.Context, slot *coursebiz.TimeSlot) error {
	return r.db.WithContext(ctx).Save(slot).Error
}

func (r *timeSlotRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&coursebiz.TimeSlot{}, "id = ?", id).Error
}

func (r *timeSlotRepository) List(ctx context.Context, schoolID string) ([]coursebiz.TimeSlot, error) {
	var slots []coursebiz.TimeSlot
	err := r.db.WithContext(ctx).Where("school_id = ?", schoolID).Order("sort_order asc").Find(&slots).Error
	return slots, err
}

func (r *timeSlotRepository) FindByID(ctx context.Context, id string) (*coursebiz.TimeSlot, error) {
	var slot coursebiz.TimeSlot
	err := r.db.WithContext(ctx).First(&slot, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, sharedbiz.ErrNotFound
	}
	return &slot, err
}
