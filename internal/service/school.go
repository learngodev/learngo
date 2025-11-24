package service

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

// SchoolService manages school operations.
type SchoolService struct {
	schools repository.SchoolRepository
}

// NewSchoolService creates a new SchoolService.
func NewSchoolService(schools repository.SchoolRepository) *SchoolService {
	return &SchoolService{schools: schools}
}

// ListSchools returns all schools.
func (s *SchoolService) ListSchools(ctx context.Context) ([]domain.School, error) {
	return s.schools.List(ctx)
}
