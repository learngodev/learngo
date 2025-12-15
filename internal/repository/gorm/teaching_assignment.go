package gormrepo

import (
	"context"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// TeachingAssignmentStore implements repository.TeachingAssignmentRepository.
type TeachingAssignmentStore struct {
	db *gorm.DB
}

// NewTeachingAssignmentStore constructs TeachingAssignmentStore.
func NewTeachingAssignmentStore(db *gorm.DB) *TeachingAssignmentStore {
	return &TeachingAssignmentStore{db: db}
}

// Create persists a new teaching assignment.
func (s *TeachingAssignmentStore) Create(ctx context.Context, assignment *domain.TeachingAssignment) error {
	return s.db.WithContext(ctx).Create(assignment).Error
}

// BatchCreate persists multiple teaching assignments.
func (s *TeachingAssignmentStore) BatchCreate(ctx context.Context, assignments []domain.TeachingAssignment) error {
	if len(assignments) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Create(&assignments).Error
}

// List returns a paginated list of teaching assignments.
func (s *TeachingAssignmentStore) List(ctx context.Context, schoolID string, courseID, teacherID, classID string, page, size int) ([]domain.TeachingAssignment, int64, error) {
	var assignments []domain.TeachingAssignment
	var total int64

	query := s.db.WithContext(ctx).Model(&domain.TeachingAssignment{}).Where("school_id = ?", schoolID)

	if courseID != "" {
		query = query.Where("course_id = ?", courseID)
	}
	if teacherID != "" {
		query = query.Where("teacher_id = ?", teacherID)
	}
	if classID != "" {
		query = query.Where("class_id = ?", classID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if size > 0 {
		offset := (page - 1) * size
		query = query.Offset(offset).Limit(size)
	}

	// Preload could be useful here if we want details, but for now just the IDs.
	// If we need details, we should probably do it in the service layer or use Preload here.
	// Let's stick to IDs for now as per interface.

	if err := query.Find(&assignments).Error; err != nil {
		return nil, 0, err
	}

	return assignments, total, nil
}

// Delete removes a teaching assignment.
func (s *TeachingAssignmentStore) Delete(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&domain.TeachingAssignment{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// BatchDelete removes multiple teaching assignments.
func (s *TeachingAssignmentStore) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Delete(&domain.TeachingAssignment{}, "id IN ?", ids).Error
}
