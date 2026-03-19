package service

import (
	"context"
	"github.com/google/uuid"
	coursebiz "learn-go/internal/biz/course"
	identitybiz "learn-go/internal/biz/identity"
	"math/rand"
	"strings"
	"time"
)

func normalizeCourseImageURL(imageURL string) string {
	return strings.TrimSpace(imageURL)
}

type CourseService struct {
	courseRepo        coursebiz.CourseRepository
	courseStudentRepo coursebiz.CourseStudentRepository
	courseTeacherRepo coursebiz.CourseTeacherRepository
	studentRepo       identitybiz.StudentRepository
	teacherRepo       identitybiz.TeacherRepository
}

func NewCourseService(
	courseRepo coursebiz.CourseRepository,
	courseStudentRepo coursebiz.CourseStudentRepository,
	courseTeacherRepo coursebiz.CourseTeacherRepository,
	studentRepo identitybiz.StudentRepository,
	teacherRepo identitybiz.TeacherRepository,
) *CourseService {
	return &CourseService{
		courseRepo:        courseRepo,
		courseStudentRepo: courseStudentRepo,
		courseTeacherRepo: courseTeacherRepo,
		studentRepo:       studentRepo,
		teacherRepo:       teacherRepo,
	}
}

func (s *CourseService) CreateCourse(ctx context.Context, schoolID string, name, description, imageURL string, classIDs []string) (*coursebiz.Course, error) {
	imageURL = normalizeCourseImageURL(imageURL)
	course := &coursebiz.Course{
		ID:             uuid.New().String(),
		SchoolID:       schoolID,
		Name:           name,
		Description:    description,
		ImageURL:       imageURL,
		InvitationCode: s.generateInvitationCode(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.courseRepo.Create(ctx, course); err != nil {
		return nil, err
	}

	if len(classIDs) > 0 {
		for _, classID := range classIDs {
			if err := s.AssignStudentsByClass(ctx, course.ID, classID); err != nil {
				return nil, err
			}
		}
	}

	return course, nil
}

func (s *CourseService) JoinCourseByCode(ctx context.Context, studentID, code string) error {
	course, err := s.courseRepo.GetByInvitationCode(ctx, code)
	if err != nil {
		return err
	}
	return s.assignStudents(ctx, course.ID, []string{studentID})
}

func (s *CourseService) generateInvitationCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func (s *CourseService) ListCourses(ctx context.Context, schoolID string, page, size int) ([]coursebiz.Course, int64, error) {
	courses, total, err := s.courseRepo.List(ctx, schoolID, page, size)
	if err != nil {
		return nil, 0, err
	}
	for i := range courses {
		courses[i].ImageURL = normalizeCourseImageURL(courses[i].ImageURL)
	}
	return courses, total, nil
}

func (s *CourseService) ListStudentCourses(ctx context.Context, studentID string) ([]coursebiz.Course, error) {
	courses, err := s.courseRepo.ListByStudentID(ctx, studentID)
	if err != nil {
		return nil, err
	}
	for i := range courses {
		courses[i].ImageURL = normalizeCourseImageURL(courses[i].ImageURL)
	}
	return courses, nil
}

func (s *CourseService) ListCourseAssignments(ctx context.Context, schoolID, courseID, departmentID, classID string, onlyAssigned bool, page, size int) ([]coursebiz.CourseAssignmentInfo, int64, error) {
	return s.courseRepo.ListAssignments(ctx, schoolID, courseID, departmentID, classID, onlyAssigned, page, size)
}

func (s *CourseService) ListCoursesWithDetails(ctx context.Context, schoolID, departmentID, classID string, page, size int) ([]coursebiz.CourseAssignmentInfo, int64, error) {
	courses, total, err := s.courseRepo.ListWithFilters(ctx, schoolID, departmentID, classID, page, size)
	if err != nil {
		return nil, 0, err
	}

	if len(courses) == 0 {
		return []coursebiz.CourseAssignmentInfo{}, total, nil
	}

	courseIDs := make([]string, len(courses))
	for i, c := range courses {
		courseIDs[i] = c.ID
	}

	assignments, err := s.courseRepo.ListAssignmentsByCourseIDs(ctx, courseIDs)
	if err != nil {
		return nil, 0, err
	}

	assignmentMap := make(map[string]*coursebiz.CourseAssignmentInfo)
	countedClasses := make(map[string]map[string]bool)

	for _, c := range courses {
		assignmentMap[c.ID] = &coursebiz.CourseAssignmentInfo{
			CourseID:    c.ID,
			CourseName:  c.Name,
			Description: c.Description,
			ImageURL:    normalizeCourseImageURL(c.ImageURL),
		}
		countedClasses[c.ID] = make(map[string]bool)
	}

	for _, a := range assignments {
		if entry, ok := assignmentMap[a.CourseID]; ok {
			entry.TeacherName = mergeUnique(entry.TeacherName, a.TeacherName)
			entry.ClassName = mergeUnique(entry.ClassName, a.ClassName)

			if a.ClassID != "" && !countedClasses[a.CourseID][a.ClassID] {
				entry.StudentCount += a.StudentCount
				countedClasses[a.CourseID][a.ClassID] = true
			}
		}
	}

	result := make([]coursebiz.CourseAssignmentInfo, len(courses))
	for i, c := range courses {
		result[i] = *assignmentMap[c.ID]
	}

	return result, total, nil
}

func mergeUnique(existing, newStr string) string {
	if newStr == "" {
		return existing
	}
	if existing == "" {
		return newStr
	}
	parts := strings.Split(existing, ", ")
	for _, p := range parts {
		if p == newStr {
			return existing
		}
	}
	return existing + ", " + newStr
}

func (s *CourseService) UpdateCourse(ctx context.Context, id, name, description, imageURL string) error {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	course.Name = name
	course.Description = description
	course.ImageURL = normalizeCourseImageURL(imageURL)
	course.UpdatedAt = time.Now()
	return s.courseRepo.Update(ctx, course)
}

func (s *CourseService) UpdateCourseFields(
	ctx context.Context,
	id string,
	name *string,
	description *string,
	imageURL *string,
) error {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	updated := false
	if name != nil {
		course.Name = strings.TrimSpace(*name)
		updated = true
	}
	if description != nil {
		course.Description = strings.TrimSpace(*description)
		updated = true
	}
	if imageURL != nil {
		course.ImageURL = normalizeCourseImageURL(*imageURL)
		updated = true
	}

	if !updated {
		return nil
	}

	course.UpdatedAt = time.Now()
	return s.courseRepo.Update(ctx, course)
}

func (s *CourseService) DeleteCourse(ctx context.Context, id string) error {
	return s.courseRepo.Delete(ctx, id)
}

func (s *CourseService) assignStudents(ctx context.Context, courseID string, studentIDs []string) error {
	if len(studentIDs) == 0 {
		return nil
	}
	enrollments := make([]coursebiz.CourseStudent, len(studentIDs))
	now := time.Now()
	for i, studentID := range studentIDs {
		enrollments[i] = coursebiz.CourseStudent{
			ID:        uuid.New().String(),
			CourseID:  courseID,
			StudentID: studentID,
			CreatedAt: now,
		}
	}
	return s.courseStudentRepo.BatchCreate(ctx, enrollments)
}

func (s *CourseService) AssignStudentsByClass(ctx context.Context, courseID string, classID string) error {
	students, err := s.studentRepo.ListByClassID(ctx, classID)
	if err != nil {
		return err
	}
	var studentIDs []string
	for _, student := range students {
		studentIDs = append(studentIDs, student.ID)
	}
	return s.assignStudents(ctx, courseID, studentIDs)
}

func (s *CourseService) AssignStudentsByDepartment(ctx context.Context, courseID string, departmentID string) error {
	students, err := s.studentRepo.ListByDepartmentID(ctx, departmentID)
	if err != nil {
		return err
	}
	var studentIDs []string
	for _, student := range students {
		studentIDs = append(studentIDs, student.ID)
	}
	return s.assignStudents(ctx, courseID, studentIDs)
}
