package gormrepo

import (
	"context"
	"strings"
	"time"

	"learn-go/internal/domain"

	"gorm.io/gorm"
)

func isMissingCourseStudentsTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if !strings.Contains(msg, "course_students") {
		return false
	}
	// PostgreSQL: pq: relation "course_students" does not exist
	// SQLite: no such table: course_students
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "no such table")
}

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

	if err := query.Preload("Teachers").Find(&courses).Error; err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

// GetByID retrieves a course by ID.
func (s *CourseStore) GetByID(ctx context.Context, id string) (*domain.Course, error) {
	var course domain.Course
	if err := s.db.WithContext(ctx).Preload("Teachers").First(&course, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &course, nil
}

// GetByInvitationCode retrieves a course by invitation code.
func (s *CourseStore) GetByInvitationCode(ctx context.Context, code string) (*domain.Course, error) {
	var course domain.Course
	if err := s.db.WithContext(ctx).First(&course, "invitation_code = ?", code).Error; err != nil {
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
func (s *CourseStore) ListAssignments(ctx context.Context, schoolID string, courseID, departmentID, classID string, onlyAssigned bool, page, size int) ([]domain.CourseAssignmentInfo, int64, error) {
	var results []domain.CourseAssignmentInfo
	var total int64

	// Base query: Join courses, teaching_assignments, teachers, accounts, classes
	// Note: We use LEFT JOIN to include courses even if they are not assigned (if no class filter is applied).
	// However, if we want to show "Assigned Teacher", we are primarily interested in assignments.
	// If the user wants "Course Management", they might want to see unassigned courses too.
	// But the requirement "Show assigned teacher and student count" implies looking at assignments.
	// If we filter by class, we definitely look at assignments.
	// If we don't filter, we might want to list all assignments.

	// Let's construct a query that selects from course_schedules primarily, joined with others.
	// We group by course, class, and teacher to avoid duplicates from multiple schedule slots.

	db := s.db.WithContext(ctx).Table("courses c").
		Select(`
			MIN(cs.id) as assignment_id,
			c.id as course_id, 
			c.name as course_name, 
			c.description, 
			c.image_url,
			t.id as teacher_id,
			a.display_name as teacher_name, 
			cl.id as class_id,
			cl.name as class_name,
			(SELECT count(*) FROM students s WHERE s.class_id = cl.id) as student_count
		`).
		Joins("LEFT JOIN course_schedules cs ON c.id = cs.course_id").
		Joins("LEFT JOIN course_teachers ct ON c.id = ct.course_id").
		Joins("LEFT JOIN teachers t ON t.id = COALESCE(cs.teacher_id, ct.teacher_id)").
		Joins("LEFT JOIN accounts a ON t.account_id = a.id").
		Joins("LEFT JOIN classes cl ON cs.class_id = cl.id").
		Where("c.school_id = ?", schoolID).
		Group("c.id, c.name, c.description, c.image_url, t.id, a.display_name, cl.id, cl.name")

	if onlyAssigned {
		db = db.Where("cs.id IS NOT NULL")
	}

	if courseID != "" {
		db = db.Where("c.id = ?", courseID)
	}
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

func (s *CourseStore) ListWithFilters(ctx context.Context, schoolID string, departmentID, classID string, page, size int) ([]domain.Course, int64, error) {
	var courses []domain.Course
	var total int64

	db := s.db.WithContext(ctx).Model(&domain.Course{}).
		Joins("LEFT JOIN course_schedules cs ON courses.id = cs.course_id").
		Joins("LEFT JOIN classes cl ON cs.class_id = cl.id").
		Where("courses.school_id = ?", schoolID)

	if departmentID != "" {
		db = db.Where("cl.department_id = ?", departmentID)
	}
	if classID != "" {
		db = db.Where("cl.id = ?", classID)
	}

	if err := db.Distinct("courses.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if size > 0 {
		offset := (page - 1) * size
		db = db.Offset(offset).Limit(size)
	}

	if err := db.Distinct("courses.*").Find(&courses).Error; err != nil {
		return nil, 0, err
	}

	return courses, total, nil
}

func (s *CourseStore) ListAssignmentsByCourseIDs(ctx context.Context, courseIDs []string) ([]domain.CourseAssignmentInfo, error) {
	var results []domain.CourseAssignmentInfo
	if len(courseIDs) == 0 {
		return results, nil
	}

	db := s.db.WithContext(ctx).Table("courses c").
		Select(`
			MIN(cs.id) as assignment_id,
			c.id as course_id, 
			c.name as course_name, 
			c.description, 
			c.image_url,
			t.id as teacher_id,
			a.display_name as teacher_name, 
			cl.id as class_id,
			cl.name as class_name,
			(SELECT count(*) FROM students s WHERE s.class_id = cl.id) as student_count
		`).
		Joins("LEFT JOIN course_schedules cs ON c.id = cs.course_id").
		Joins("LEFT JOIN course_teachers ct ON c.id = ct.course_id").
		Joins("LEFT JOIN teachers t ON t.id = COALESCE(cs.teacher_id, ct.teacher_id)").
		Joins("LEFT JOIN accounts a ON t.account_id = a.id").
		Joins("LEFT JOIN classes cl ON cs.class_id = cl.id").
		Where("c.id IN ?", courseIDs).
		Group("c.id, c.name, c.description, c.image_url, t.id, a.display_name, cl.id, cl.name")

	if err := db.Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

func (s *CourseStore) ListByStudentID(ctx context.Context, studentID string) ([]domain.Course, error) {
	var courses []domain.Course
	err := s.db.WithContext(ctx).
		Model(&domain.Course{}).
		Joins("JOIN course_students ON course_students.course_id = courses.id").
		Where("course_students.student_id = ?", studentID).
		Find(&courses).Error
	if err == nil && len(courses) > 0 {
		return courses, nil
	}
	if err != nil && !isMissingCourseStudentsTable(err) {
		return nil, err
	}

	// Fallback: infer courses from the student's class schedule.
	// This keeps sample databases (which may not seed enrollments) usable.
	courses = nil
	if err2 := s.db.WithContext(ctx).
		Model(&domain.Course{}).
		Distinct("courses.*").
		Joins("JOIN course_schedules cs ON cs.course_id = courses.id").
		Joins("JOIN students st ON st.class_id = cs.class_id").
		Where("st.id = ?", studentID).
		Find(&courses).Error; err2 != nil {
		return nil, err2
	}

	return courses, nil
}

// CourseSessionStore implements repository.CourseSessionRepository.
type CourseSessionStore struct {
	db *gorm.DB
}

// NewCourseSessionStore constructs CourseSessionStore.
func NewCourseSessionStore(db *gorm.DB) *CourseSessionStore {
	return &CourseSessionStore{db: db}
}

// ListBetween returns sessions within the provided time window.
func (s *CourseSessionStore) ListBetween(ctx context.Context, start, end time.Time) ([]domain.CourseSession, error) {
	query := s.db.WithContext(ctx)
	if !end.IsZero() {
		query = query.Where("starts_at < ?", end)
	}
	if !start.IsZero() {
		query = query.Where("ends_at > ?", start)
	}
	query = query.Order("starts_at ASC")

	var sessions []domain.CourseSession
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
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

func (s *CourseSessionStore) Create(ctx context.Context, session *domain.CourseSession) error {
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *CourseSessionStore) Exists(ctx context.Context, courseID, classID string, teacherID *string, slotID string, startsAt time.Time) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&domain.CourseSession{}).
		Where("course_id = ? AND class_id = ? AND teacher_id = ? AND slot_id = ? AND starts_at = ?",
			courseID, classID, teacherID, slotID, startsAt).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *CourseSessionStore) Update(ctx context.Context, session *domain.CourseSession) error {
	return s.db.WithContext(ctx).Save(session).Error
}

func (s *CourseSessionStore) DeleteByIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&domain.CourseSession{}).Error
}

func (s *CourseSessionStore) GetByID(ctx context.Context, id string) (*domain.CourseSession, error) {
	var session domain.CourseSession
	if err := s.db.WithContext(ctx).First(&session, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &session, nil
}
