package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"

	"gorm.io/gorm"
)

// CourseStore implements repository.CourseRepository.
type CourseStore struct {
	db *gorm.DB
}

// NewCourseStore constructs CourseStore.
func NewCourseStore(db *gorm.DB) *CourseStore {
	return &CourseStore{db: db}
}

// Create persists a new course.
func (s *CourseStore) Create(ctx context.Context, course *domain.Course) error {
	return s.db.WithContext(ctx).Create(course).Error
}

// List returns a paginated list of courses for a school.
func (s *CourseStore) List(ctx context.Context, schoolID string, page, size int) ([]domain.Course, int64, error) {
	var courses []domain.Course
	var total int64

	query := s.db.WithContext(ctx).Model(&domain.Course{}).Where("school_id = ?", schoolID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if size > 0 {
		offset := (page - 1) * size
		query = query.Offset(offset).Limit(size)
	}

	if err := query.Find(&courses).Error; err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

// GetByID retrieves a course by ID.
func (s *CourseStore) GetByID(ctx context.Context, id string) (*domain.Course, error) {
	var course domain.Course
	if err := s.db.WithContext(ctx).First(&course, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &course, nil
}

// Update modifies an existing course.
func (s *CourseStore) Update(ctx context.Context, course *domain.Course) error {
	return s.db.WithContext(ctx).Save(course).Error
}

// Delete removes a course.
func (s *CourseStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&domain.Course{}, "id = ?", id).Error
}

// ListByIDs returns course metadata for the provided identifiers.
func (s *CourseStore) ListByIDs(ctx context.Context, ids []string) ([]domain.Course, error) {
	if len(ids) == 0 {
		return []domain.Course{}, nil
	}

	var courses []domain.Course
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}

// ListAssignments returns enriched course assignment data.
func (s *CourseStore) ListAssignments(ctx context.Context, schoolID string, departmentID, classID string, page, size int) ([]domain.CourseAssignmentInfo, int64, error) {
	var results []domain.CourseAssignmentInfo
	var total int64

	// Base query: Join courses, teaching_assignments, teachers, accounts, classes
	// Note: We use LEFT JOIN to include courses even if they are not assigned (if no class filter is applied).
	// However, if we want to show "Assigned Teacher", we are primarily interested in assignments.
	// If the user wants "Course Management", they might want to see unassigned courses too.
	// But the requirement "Show assigned teacher and student count" implies looking at assignments.
	// If we filter by class, we definitely look at assignments.
	// If we don't filter, we might want to list all assignments.

	// Let's construct a query that selects from teaching_assignments primarily, joined with others.
	// But wait, if a course is not assigned, it won't be in teaching_assignments.
	// If the user wants to see "Total Courses", they expect to see unassigned ones too.
	// So we should select from courses and LEFT JOIN assignments.

	db := s.db.WithContext(ctx).Table("courses c").
		Select(`
			c.id as course_id, 
			c.name as course_name, 
			c.description, 
			t.id as teacher_id,
			a.display_name as teacher_name, 
			cl.id as class_id,
			cl.name as class_name,
			(SELECT count(*) FROM students s WHERE s.class_id = cl.id) as student_count
		`).
		Joins("LEFT JOIN teaching_assignments ta ON c.id = ta.course_id").
		Joins("LEFT JOIN teachers t ON ta.teacher_id = t.id").
		Joins("LEFT JOIN accounts a ON t.account_id = a.id").
		Joins("LEFT JOIN classes cl ON ta.class_id = cl.id").
		Where("c.school_id = ?", schoolID)

	if departmentID != "" {
		db = db.Where("cl.department_id = ?", departmentID)
	}
	if classID != "" {
		db = db.Where("cl.id = ?", classID)
	}

	// Count total rows (assignments + unassigned courses)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if size > 0 {
		offset := (page - 1) * size
		db = db.Offset(offset).Limit(size)
	}

	if err := db.Scan(&results).Error; err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// CourseSessionStore implements repository.CourseSessionRepository.
type CourseSessionStore struct {
	db *gorm.DB
}

// NewCourseSessionStore constructs CourseSessionStore.
func NewCourseSessionStore(db *gorm.DB) *CourseSessionStore {
	return &CourseSessionStore{db: db}
}

// ListByClassBetween returns sessions for a class within the provided time window.
func (s *CourseSessionStore) ListByClassBetween(ctx context.Context, classID string, start, end time.Time) ([]domain.CourseSession, error) {
	query := s.db.WithContext(ctx).Where("class_id = ?", classID)
	if !start.IsZero() {
		query = query.Where("starts_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("starts_at < ?", end)
	}
	query = query.Order("starts_at ASC")

	var sessions []domain.CourseSession
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// ListByTeacherBetween returns sessions taught by a teacher within the time window.
func (s *CourseSessionStore) ListByTeacherBetween(ctx context.Context, teacherID string, start, end time.Time) ([]domain.CourseSession, error) {
	query := s.db.WithContext(ctx).Where("teacher_id = ?", teacherID)
	if !start.IsZero() {
		query = query.Where("starts_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("starts_at < ?", end)
	}
	query = query.Order("starts_at ASC")

	var sessions []domain.CourseSession
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}
