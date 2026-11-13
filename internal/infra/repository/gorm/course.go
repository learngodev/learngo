package gormrepo

import (
	"context"
	"strings"
	"time"

	"learn-go/internal/domain/curriculum"
	"learn-go/internal/domain/scheduling"

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
	// PostgreSQL：pq: relation "course_students" does not exist
	// SQLite：no such table: course_students
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "no such table")
}

// CourseStore 实现 curriculum.CourseRepository。
type CourseStore struct {
	db *gorm.DB
}

// NewCourseStore 构造 CourseStore。
func NewCourseStore(db *gorm.DB) *CourseStore {
	return &CourseStore{db: db}
}

// Create 持久化一门新课程。
func (s *CourseStore) Create(ctx context.Context, course *curriculum.Course) error {
	return s.db.WithContext(ctx).Create(course).Error
}

// List 分页返回某学校的课程列表。
func (s *CourseStore) List(ctx context.Context, schoolID string, page, size int) ([]curriculum.Course, int64, error) {
	var courses []curriculum.Course
	var total int64

	query := s.db.WithContext(ctx).Model(&curriculum.Course{}).Where("school_id = ?", schoolID)

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

// GetByID 按 ID 查询课程。
func (s *CourseStore) GetByID(ctx context.Context, id string) (*curriculum.Course, error) {
	var course curriculum.Course
	if err := s.db.WithContext(ctx).Preload("Teachers").First(&course, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	return &course, nil
}

// GetByInvitationCode 按邀请码查询课程。
func (s *CourseStore) GetByInvitationCode(ctx context.Context, code string) (*curriculum.Course, error) {
	var course curriculum.Course
	if err := s.db.WithContext(ctx).First(&course, "invitation_code = ?", code).Error; err != nil {
		return nil, notFound(err)
	}
	return &course, nil
}

// Update 修改已有课程。
func (s *CourseStore) Update(ctx context.Context, course *curriculum.Course) error {
	return s.db.WithContext(ctx).Save(course).Error
}

// Delete 删除课程。
func (s *CourseStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Delete(&curriculum.Course{}, "id = ?", id).Error
}

// ListByIDs 返回指定 ID 对应的课程信息。
func (s *CourseStore) ListByIDs(ctx context.Context, ids []string) ([]curriculum.Course, error) {
	if len(ids) == 0 {
		return []curriculum.Course{}, nil
	}

	var courses []curriculum.Course
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Find(&courses).Error; err != nil {
		return nil, err
	}
	return courses, nil
}

// ListAssignments 返回补充了关联信息的课程分配数据。
func (s *CourseStore) ListAssignments(ctx context.Context, schoolID string, courseID, departmentID, classID string, onlyAssigned bool, page, size int) ([]curriculum.CourseAssignmentInfo, int64, error) {
	var results []curriculum.CourseAssignmentInfo
	var total int64

	// 基础查询：关联 courses、teaching_assignments、teachers、accounts、classes
	// 注意：使用 LEFT JOIN，以便在未按班级筛选时也包含尚未分配的课程。
	// 但若要展示“授课教师”，我们主要关心的是分配记录。
	// 在“课程管理”场景下，用户可能也希望看到未分配的课程。
	// 但“显示授课教师和学生人数”的需求意味着要查看分配记录。
	// 按班级筛选时，必然查询分配记录。
	// 不筛选时，可能需要列出所有分配记录。

	// 因此以 course_schedules 为主表构造查询，并关联其他表。
	// 按课程、班级、教师分组，避免多个排课节次导致重复。

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

	// 统计总行数（分配记录 + 未分配课程）
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

func (s *CourseStore) ListWithFilters(ctx context.Context, schoolID string, departmentID, classID string, page, size int) ([]curriculum.Course, int64, error) {
	var courses []curriculum.Course
	var total int64

	db := s.db.WithContext(ctx).Model(&curriculum.Course{}).
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

func (s *CourseStore) ListAssignmentsByCourseIDs(ctx context.Context, courseIDs []string) ([]curriculum.CourseAssignmentInfo, error) {
	var results []curriculum.CourseAssignmentInfo
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

func (s *CourseStore) ListByStudentID(ctx context.Context, studentID string) ([]curriculum.Course, error) {
	var courses []curriculum.Course
	err := s.db.WithContext(ctx).
		Model(&curriculum.Course{}).
		Joins("JOIN course_students ON course_students.course_id = courses.id").
		Where("course_students.student_id = ?", studentID).
		Find(&courses).Error
	if err == nil && len(courses) > 0 {
		return courses, nil
	}
	if err != nil && !isMissingCourseStudentsTable(err) {
		return nil, err
	}

	// 回退：根据学生所在班级的课表推断课程。
	// 这样示例数据库（可能未预置选课数据）也能正常使用。
	courses = nil
	if err2 := s.db.WithContext(ctx).
		Model(&curriculum.Course{}).
		Distinct("courses.*").
		Joins("JOIN course_schedules cs ON cs.course_id = courses.id").
		Joins("JOIN students st ON st.class_id = cs.class_id").
		Where("st.id = ?", studentID).
		Find(&courses).Error; err2 != nil {
		return nil, err2
	}

	return courses, nil
}

// CourseSessionStore 实现 scheduling.CourseSessionRepository。
type CourseSessionStore struct {
	db *gorm.DB
}

// NewCourseSessionStore 构造 CourseSessionStore。
func NewCourseSessionStore(db *gorm.DB) *CourseSessionStore {
	return &CourseSessionStore{db: db}
}

// ListBetween 返回指定时间窗口内的课次。
func (s *CourseSessionStore) ListBetween(ctx context.Context, start, end time.Time) ([]scheduling.CourseSession, error) {
	query := s.db.WithContext(ctx)
	if !end.IsZero() {
		query = query.Where("starts_at < ?", end)
	}
	if !start.IsZero() {
		query = query.Where("ends_at > ?", start)
	}
	query = query.Order("starts_at ASC")

	var sessions []scheduling.CourseSession
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// ListByClassBetween 返回某班级在指定时间窗口内的课次。
func (s *CourseSessionStore) ListByClassBetween(ctx context.Context, classID string, start, end time.Time) ([]scheduling.CourseSession, error) {
	query := s.db.WithContext(ctx).Where("class_id = ?", classID)
	if !start.IsZero() {
		query = query.Where("starts_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("starts_at < ?", end)
	}
	query = query.Order("starts_at ASC")

	var sessions []scheduling.CourseSession
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// ListByTeacherBetween 返回某教师在指定时间窗口内的授课课次。
func (s *CourseSessionStore) ListByTeacherBetween(ctx context.Context, teacherID string, start, end time.Time) ([]scheduling.CourseSession, error) {
	query := s.db.WithContext(ctx).Where("teacher_id = ?", teacherID)
	if !start.IsZero() {
		query = query.Where("starts_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("starts_at < ?", end)
	}
	query = query.Order("starts_at ASC")

	var sessions []scheduling.CourseSession
	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *CourseSessionStore) Create(ctx context.Context, session *scheduling.CourseSession) error {
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *CourseSessionStore) Exists(ctx context.Context, courseID, classID string, teacherID *string, slotID string, startsAt time.Time) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&scheduling.CourseSession{}).
		Where("course_id = ? AND class_id = ? AND teacher_id = ? AND slot_id = ? AND starts_at = ?",
			courseID, classID, teacherID, slotID, startsAt).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *CourseSessionStore) Update(ctx context.Context, session *scheduling.CourseSession) error {
	return s.db.WithContext(ctx).Save(session).Error
}

func (s *CourseSessionStore) DeleteByIDs(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&scheduling.CourseSession{}).Error
}

func (s *CourseSessionStore) GetByID(ctx context.Context, id string) (*scheduling.CourseSession, error) {
	var session scheduling.CourseSession
	if err := s.db.WithContext(ctx).First(&session, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	return &session, nil
}

var (
	_ curriculum.CourseRepository        = (*CourseStore)(nil)
	_ scheduling.CourseSessionRepository = (*CourseSessionStore)(nil)
)
