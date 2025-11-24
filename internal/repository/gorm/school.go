package gormrepo

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// SchoolStore implements repository.SchoolRepository using GORM.
type SchoolStore struct {
	db *gorm.DB
}

// NewSchoolStore returns a new school store.
func NewSchoolStore(db *gorm.DB) *SchoolStore {
	return &SchoolStore{db: db}
}

func (s *SchoolStore) List(ctx context.Context) ([]domain.School, error) {
	var schools []domain.School
	if err := s.db.WithContext(ctx).Order("created_at").Find(&schools).Error; err != nil {
		return nil, err
	}
	return schools, nil
}

var _ repository.SchoolRepository = (*SchoolStore)(nil)
