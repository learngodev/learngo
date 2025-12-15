package gormrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

type timeSlotRepository struct {
	db *gorm.DB
}

func NewTimeSlotRepository(db *gorm.DB) repository.TimeSlotRepository {
	return &timeSlotRepository{db: db}
}

func (r *timeSlotRepository) Create(ctx context.Context, slot *domain.TimeSlot) error {
	return r.db.WithContext(ctx).Create(slot).Error
}

func (r *timeSlotRepository) Update(ctx context.Context, slot *domain.TimeSlot) error {
	return r.db.WithContext(ctx).Save(slot).Error
}

func (r *timeSlotRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.TimeSlot{}, "id = ?", id).Error
}

func (r *timeSlotRepository) List(ctx context.Context, schoolID string) ([]domain.TimeSlot, error) {
	var slots []domain.TimeSlot
	err := r.db.WithContext(ctx).Where("school_id = ?", schoolID).Order("sort_order asc").Find(&slots).Error
	return slots, err
}

func (r *timeSlotRepository) FindByID(ctx context.Context, id string) (*domain.TimeSlot, error) {
	var slot domain.TimeSlot
	err := r.db.WithContext(ctx).First(&slot, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	return &slot, err
}
