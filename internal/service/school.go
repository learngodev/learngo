package service

import (
	"context"

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

// CreateTimeSlot creates a new time slot.
func (s *SchoolService) CreateTimeSlot(ctx context.Context, slot *domain.TimeSlot) error {
	return s.timeSlots.Create(ctx, slot)
}

// UpdateTimeSlot updates an existing time slot.
func (s *SchoolService) UpdateTimeSlot(ctx context.Context, slot *domain.TimeSlot) error {
	return s.timeSlots.Update(ctx, slot)
}

// DeleteTimeSlot deletes a time slot.
func (s *SchoolService) DeleteTimeSlot(ctx context.Context, id string) error {
	return s.timeSlots.Delete(ctx, id)
}
