package usecase

import (
	"context"
	"errors"
	"time"

	"learn-go/internal/domain/organization"
	"learn-go/internal/domain/scheduling"
)

// SchoolUsecase 负责学校相关操作。
type SchoolUsecase struct {
	schools   organization.SchoolRepository
	timeSlots scheduling.TimeSlotRepository
}

// NewSchoolUsecase 创建 SchoolUsecase。
func NewSchoolUsecase(schools organization.SchoolRepository, timeSlots scheduling.TimeSlotRepository) *SchoolUsecase {
	return &SchoolUsecase{schools: schools, timeSlots: timeSlots}
}

// ListSchools 返回所有学校。
func (s *SchoolUsecase) ListSchools(ctx context.Context) ([]organization.School, error) {
	return s.schools.List(ctx)
}

// ListTimeSlots 返回某学校的所有节次。
func (s *SchoolUsecase) ListTimeSlots(ctx context.Context, schoolID string) ([]scheduling.TimeSlot, error) {
	return s.timeSlots.List(ctx, schoolID)
}

// GetTimeSlot 按 ID 返回节次。
func (s *SchoolUsecase) GetTimeSlot(ctx context.Context, id string) (*scheduling.TimeSlot, error) {
	return s.timeSlots.FindByID(ctx, id)
}

// CreateTimeSlot 创建新节次。
func (s *SchoolUsecase) CreateTimeSlot(ctx context.Context, slot *scheduling.TimeSlot) error {
	if err := s.validateTimeSlot(slot); err != nil {
		return err
	}
	return s.timeSlots.Create(ctx, slot)
}

// UpdateTimeSlot 更新已有节次。
func (s *SchoolUsecase) UpdateTimeSlot(ctx context.Context, slot *scheduling.TimeSlot) error {
	if err := s.validateTimeSlot(slot); err != nil {
		return err
	}
	return s.timeSlots.Update(ctx, slot)
}

// DeleteTimeSlot 删除节次。
func (s *SchoolUsecase) DeleteTimeSlot(ctx context.Context, id string) error {
	return s.timeSlots.Delete(ctx, id)
}

func (s *SchoolUsecase) validateTimeSlot(slot *scheduling.TimeSlot) error {
	start, err := time.Parse("15:04", slot.StartTime)
	if err != nil {
		return errors.New("invalid start_time format, expected HH:mm")
	}
	end, err := time.Parse("15:04", slot.EndTime)
	if err != nil {
		return errors.New("invalid end_time format, expected HH:mm")
	}
	if !start.Before(end) {
		return errors.New("start_time must be before end_time")
	}
	return nil
}
