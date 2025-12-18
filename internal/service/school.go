package service

import (
	"context"
	"errors"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

// SchoolService manages school operations.
type SchoolService struct {
	schools   repository.SchoolRepository
	timeSlots repository.TimeSlotRepository
}

// NewSchoolService creates a new SchoolService.
func NewSchoolService(schools repository.SchoolRepository, timeSlots repository.TimeSlotRepository) *SchoolService {
	return &SchoolService{schools: schools, timeSlots: timeSlots}
}

// ListSchools returns all schools.
func (s *SchoolService) ListSchools(ctx context.Context) ([]domain.School, error) {
	return s.schools.List(ctx)
}

// ListTimeSlots returns all time slots for a school.
func (s *SchoolService) ListTimeSlots(ctx context.Context, schoolID string) ([]domain.TimeSlot, error) {
	return s.timeSlots.List(ctx, schoolID)
}

// GetTimeSlot returns a time slot by ID.
func (s *SchoolService) GetTimeSlot(ctx context.Context, id string) (*domain.TimeSlot, error) {
	return s.timeSlots.FindByID(ctx, id)
}

// CreateTimeSlot creates a new time slot.
func (s *SchoolService) CreateTimeSlot(ctx context.Context, slot *domain.TimeSlot) error {
	if err := s.validateTimeSlot(slot); err != nil {
		return err
	}
	return s.timeSlots.Create(ctx, slot)
}

// UpdateTimeSlot updates an existing time slot.
func (s *SchoolService) UpdateTimeSlot(ctx context.Context, slot *domain.TimeSlot) error {
	if err := s.validateTimeSlot(slot); err != nil {
		return err
	}
	return s.timeSlots.Update(ctx, slot)
}

// DeleteTimeSlot deletes a time slot.
func (s *SchoolService) DeleteTimeSlot(ctx context.Context, id string) error {
	return s.timeSlots.Delete(ctx, id)
}

func (s *SchoolService) validateTimeSlot(slot *domain.TimeSlot) error {
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
