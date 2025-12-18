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

// ListDetails returns a paginated list of teaching assignments with related entity names.
func (s *TeachingAssignmentStore) ListDetails(ctx context.Context, schoolID string, courseID, teacherID, classID string, page, size int) ([]domain.TeachingAssignmentDetail, int64, error) {
	var assignments []domain.TeachingAssignmentDetail
	var total int64

	query := s.db.WithContext(ctx).Table("teaching_assignments").
		Select("teaching_assignments.*, classes.name as class_name, accounts.display_name as teacher_name, courses.name as course_name, (SELECT count(*) FROM students WHERE students.class_id = teaching_assignments.class_id) as student_count").
		Joins("LEFT JOIN classes ON classes.id = teaching_assignments.class_id").
		Joins("LEFT JOIN teachers ON teachers.id = teaching_assignments.teacher_id").
		Joins("LEFT JOIN accounts ON accounts.id = teachers.account_id").
		Joins("LEFT JOIN courses ON courses.id = teaching_assignments.course_id").
		Where("teaching_assignments.school_id = ?", schoolID)

	if courseID != "" {
		query = query.Where("teaching_assignments.course_id = ?", courseID)
	}
	if teacherID != "" {
		query = query.Where("teaching_assignments.teacher_id = ?", teacherID)
	}
	if classID != "" {
		query = query.Where("teaching_assignments.class_id = ?", classID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if size > 0 {
		offset := (page - 1) * size
		query = query.Offset(offset).Limit(size)
	}

	if err := query.Scan(&assignments).Error; err != nil {
		return nil, 0, err
	}

	return assignments, total, nil
}

// GetByID retrieves a teaching assignment by ID.
func (s *TeachingAssignmentStore) GetByID(ctx context.Context, id string) (*domain.TeachingAssignment, error) {
	var assignment domain.TeachingAssignment
	if err := s.db.WithContext(ctx).First(&assignment, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &assignment, nil
}

// Update modifies an existing teaching assignment.
func (s *TeachingAssignmentStore) Update(ctx context.Context, assignment *domain.TeachingAssignment) error {
	return s.db.WithContext(ctx).Save(assignment).Error
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
