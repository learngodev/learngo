package gormrepo

import (
	"context"

	"learn-go/internal/domain/organization"

	"gorm.io/gorm"
)

// SchoolStore 基于 GORM 实现 organization.SchoolRepository。
type SchoolStore struct {
	db *gorm.DB
}

// NewSchoolStore 返回新的学校存储。
func NewSchoolStore(db *gorm.DB) *SchoolStore {
	return &SchoolStore{db: db}
}

func (s *SchoolStore) List(ctx context.Context) ([]organization.School, error) {
	var schools []organization.School
	if err := s.db.WithContext(ctx).Order("created_at").Find(&schools).Error; err != nil {
		return nil, err
	}
	return schools, nil
}

var _ organization.SchoolRepository = (*SchoolStore)(nil)
