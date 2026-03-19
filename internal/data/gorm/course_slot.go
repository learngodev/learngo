package gormrepo

import (
	"context"
	"gorm.io/gorm"
	coursebiz "learn-go/internal/biz/course"
)

// CourseSlotStore implements coursebiz.CourseSlotRepository.
type CourseSlotStore struct {
	db *gorm.DB
}

// NewCourseSlotStore constructs CourseSlotStore.
func NewCourseSlotStore(db *gorm.DB) *CourseSlotStore {
	return &CourseSlotStore{db: db}
}

// ListByIDs fetches course slots by identifiers.
func (s *CourseSlotStore) ListByIDs(ctx context.Context, ids []string) ([]coursebiz.CourseSlot, error) {
	if len(ids) == 0 {
		return []coursebiz.CourseSlot{}, nil
	}

	var slots []coursebiz.CourseSlot
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&slots).Error; err != nil {
		return nil, err
	}
	return slots, nil
}
