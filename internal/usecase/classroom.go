package usecase

import (
	"context"
	"time"

	"learn-go/internal/domain/organization"

	"github.com/google/uuid"
)

type ClassroomUsecase struct {
	classroomRepo organization.ClassroomRepository
}

func NewClassroomUsecase(classroomRepo organization.ClassroomRepository) *ClassroomUsecase {
	return &ClassroomUsecase{classroomRepo: classroomRepo}
}

func (s *ClassroomUsecase) Create(ctx context.Context, schoolID, location string) (*organization.Classroom, error) {
	classroom := &organization.Classroom{
		ID:        uuid.New().String(),
		SchoolID:  schoolID,
		Location:  location,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.classroomRepo.Create(ctx, classroom); err != nil {
		return nil, err
	}
	return classroom, nil
}

func (s *ClassroomUsecase) List(ctx context.Context, schoolID string, page, size int) ([]organization.Classroom, int64, error) {
	return s.classroomRepo.List(ctx, schoolID, page, size)
}

func (s *ClassroomUsecase) Update(ctx context.Context, id, location string) (*organization.Classroom, error) {
	classroom, err := s.classroomRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	classroom.Location = location
	classroom.UpdatedAt = time.Now()
	if err := s.classroomRepo.Update(ctx, classroom); err != nil {
		return nil, err
	}
	return classroom, nil
}

func (s *ClassroomUsecase) Delete(ctx context.Context, id string) error {
	return s.classroomRepo.Delete(ctx, id)
}
