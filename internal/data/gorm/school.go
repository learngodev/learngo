package gormrepo

import (
	"context"
	"gorm.io/gorm"
	schoolbiz "learn-go/internal/biz/school"
)

// SchoolStore implements schoolbiz.SchoolRepository using GORM.
type SchoolStore struct {
	db *gorm.DB
}

// NewSchoolStore returns a new school store.
func NewSchoolStore(db *gorm.DB) *SchoolStore {
	return &SchoolStore{db: db}
}

func (s *SchoolStore) List(ctx context.Context) ([]schoolbiz.School, error) {
	var schools []schoolbiz.School
	if err := s.db.WithContext(ctx).Order("created_at").Find(&schools).Error; err != nil {
		return nil, err
	}
	return schools, nil
}

var _ schoolbiz.SchoolRepository = (*SchoolStore)(nil)
