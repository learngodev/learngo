package usecase

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"learn-go/internal/domain/curriculum"
	"learn-go/internal/domain/roster"

	"github.com/google/uuid"
)

func normalizeCourseImageURL(imageURL string) string {
	return strings.TrimSpace(imageURL)
}

type CourseUsecase struct {
	courseRepo        curriculum.CourseRepository
	courseStudentRepo curriculum.CourseStudentRepository
	courseTeacherRepo curriculum.CourseTeacherRepository
	studentRepo       roster.StudentRepository
	teacherRepo       roster.TeacherRepository
}

func NewCourseUsecase(
	courseRepo curriculum.CourseRepository,
	courseStudentRepo curriculum.CourseStudentRepository,
	courseTeacherRepo curriculum.CourseTeacherRepository,
	studentRepo roster.StudentRepository,
	teacherRepo roster.TeacherRepository,
) *CourseUsecase {
	return &CourseUsecase{
		courseRepo:        courseRepo,
		courseStudentRepo: courseStudentRepo,
		courseTeacherRepo: courseTeacherRepo,
		studentRepo:       studentRepo,
		teacherRepo:       teacherRepo,
	}
}

func (s *CourseUsecase) CreateCourse(ctx context.Context, schoolID string, name, description, imageURL string, classIDs []string) (*curriculum.Course, error) {
	imageURL = normalizeCourseImageURL(imageURL)
	course := &curriculum.Course{
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

func (s *CourseUsecase) JoinCourseByCode(ctx context.Context, studentID, code string) error {
	course, err := s.courseRepo.GetByInvitationCode(ctx, code)
	if err != nil {
		return err
	}
	return s.assignStudents(ctx, course.ID, []string{studentID})
}

func (s *CourseUsecase) generateInvitationCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func (s *CourseUsecase) ListCourses(ctx context.Context, schoolID string, page, size int) ([]curriculum.Course, int64, error) {
	courses, total, err := s.courseRepo.List(ctx, schoolID, page, size)
	if err != nil {
		return nil, 0, err
	}
	for i := range courses {
		courses[i].ImageURL = normalizeCourseImageURL(courses[i].ImageURL)
	}
	return courses, total, nil
}

func (s *CourseUsecase) ListStudentCourses(ctx context.Context, studentID string) ([]curriculum.Course, error) {
	courses, err := s.courseRepo.ListByStudentID(ctx, studentID)
	if err != nil {
		return nil, err
	}
	for i := range courses {
		courses[i].ImageURL = normalizeCourseImageURL(courses[i].ImageURL)
	}
	return courses, nil
}

func (s *CourseUsecase) ListCourseAssignments(ctx context.Context, schoolID, courseID, departmentID, classID string, onlyAssigned bool, page, size int) ([]curriculum.CourseAssignmentInfo, int64, error) {
	return s.courseRepo.ListAssignments(ctx, schoolID, courseID, departmentID, classID, onlyAssigned, page, size)
}

func (s *CourseUsecase) ListCoursesWithDetails(ctx context.Context, schoolID, departmentID, classID string, page, size int) ([]curriculum.CourseAssignmentInfo, int64, error) {
	courses, total, err := s.courseRepo.ListWithFilters(ctx, schoolID, departmentID, classID, page, size)
	if err != nil {
		return nil, 0, err
	}

	if len(courses) == 0 {
		return []curriculum.CourseAssignmentInfo{}, total, nil
	}

	courseIDs := make([]string, len(courses))
	for i, c := range courses {
		courseIDs[i] = c.ID
	}

	assignments, err := s.courseRepo.ListAssignmentsByCourseIDs(ctx, courseIDs)
	if err != nil {
		return nil, 0, err
	}

	assignmentMap := make(map[string]*curriculum.CourseAssignmentInfo)
	countedClasses := make(map[string]map[string]bool)

	for _, c := range courses {
		assignmentMap[c.ID] = &curriculum.CourseAssignmentInfo{
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

	result := make([]curriculum.CourseAssignmentInfo, len(courses))
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

func (s *CourseUsecase) UpdateCourse(ctx context.Context, id, name, description, imageURL string) error {
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

func (s *CourseUsecase) UpdateCourseFields(
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

func (s *CourseUsecase) DeleteCourse(ctx context.Context, id string) error {
	return s.courseRepo.Delete(ctx, id)
}

func (s *CourseUsecase) assignStudents(ctx context.Context, courseID string, studentIDs []string) error {
	if len(studentIDs) == 0 {
		return nil
	}
	enrollments := make([]curriculum.CourseStudent, len(studentIDs))
	now := time.Now()
	for i, studentID := range studentIDs {
		enrollments[i] = curriculum.CourseStudent{
			ID:        uuid.New().String(),
			CourseID:  courseID,
			StudentID: studentID,
			CreatedAt: now,
		}
	}
	return s.courseStudentRepo.BatchCreate(ctx, enrollments)
}

func (s *CourseUsecase) AssignStudentsByClass(ctx context.Context, courseID string, classID string) error {
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

func (s *CourseUsecase) AssignStudentsByDepartment(ctx context.Context, courseID string, departmentID string) error {
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
