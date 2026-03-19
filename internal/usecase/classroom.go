package service

import (
	"context"
	"github.com/google/uuid"
	coursebiz "learn-go/internal/biz/course"
	"time"
)

type ClassroomService struct {
	classroomRepo coursebiz.ClassroomRepository
}

func NewClassroomService(classroomRepo coursebiz.ClassroomRepository) *ClassroomService {
	return &ClassroomService{classroomRepo: classroomRepo}
}

func (s *ClassroomService) Create(ctx context.Context, schoolID, location string) (*coursebiz.Classroom, error) {
	classroom := &coursebiz.Classroom{
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

func (s *ClassroomService) List(ctx context.Context, schoolID string, page, size int) ([]coursebiz.Classroom, int64, error) {
	return s.classroomRepo.List(ctx, schoolID, page, size)
}

func (s *ClassroomService) Update(ctx context.Context, id, location string) (*coursebiz.Classroom, error) {
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

func (s *ClassroomService) Delete(ctx context.Context, id string) error {
	return s.classroomRepo.Delete(ctx, id)
}
