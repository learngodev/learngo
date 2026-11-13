package gormrepo

import (
	"context"

	"learn-go/internal/domain/scheduling"

	"gorm.io/gorm"
)

// CourseSlotStore 实现 scheduling.CourseSlotRepository。
type CourseSlotStore struct {
	db *gorm.DB
}

// NewCourseSlotStore 构造 CourseSlotStore。
func NewCourseSlotStore(db *gorm.DB) *CourseSlotStore {
	return &CourseSlotStore{db: db}
}

// ListByIDs 按 ID 查询课程节次。
func (s *CourseSlotStore) ListByIDs(ctx context.Context, ids []string) ([]scheduling.CourseSlot, error) {
	if len(ids) == 0 {
		return []scheduling.CourseSlot{}, nil
	}

	var slots []scheduling.CourseSlot
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&slots).Error; err != nil {
		return nil, err
	}
	return slots, nil
}

var _ scheduling.CourseSlotRepository = (*CourseSlotStore)(nil)
