package gormrepo

import (
	"context"

	"learn-go/internal/domain"

	"gorm.io/gorm"
)

// CourseSlotStore implements repository.CourseSlotRepository.
type CourseSlotStore struct {
	db *gorm.DB
}

// NewCourseSlotStore constructs CourseSlotStore.
func NewCourseSlotStore(db *gorm.DB) *CourseSlotStore {
	return &CourseSlotStore{db: db}
}

// ListByIDs fetches course slots by identifiers.
func (s *CourseSlotStore) ListByIDs(ctx context.Context, ids []string) ([]domain.CourseSlot, error) {
	if len(ids) == 0 {
		return []domain.CourseSlot{}, nil
	}

	var slots []domain.CourseSlot
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&slots).Error; err != nil {
		return nil, err
	}
	return slots, nil
}
