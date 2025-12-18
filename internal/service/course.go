package service

import (
	"context"
	"errors"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"github.com/google/uuid"
)

type CourseService struct {
	courseRepo             repository.CourseRepository
	teachingAssignmentRepo repository.TeachingAssignmentRepository
	courseStudentRepo      repository.CourseStudentRepository
	studentRepo            repository.StudentRepository
	teacherRepo            repository.TeacherRepository
}

func NewCourseService(
	courseRepo repository.CourseRepository,
	teachingAssignmentRepo repository.TeachingAssignmentRepository,
	courseStudentRepo repository.CourseStudentRepository,
	studentRepo repository.StudentRepository,
	teacherRepo repository.TeacherRepository,
) *CourseService {
	return &CourseService{
		courseRepo:             courseRepo,
		teachingAssignmentRepo: teachingAssignmentRepo,
		courseStudentRepo:      courseStudentRepo,
		studentRepo:            studentRepo,
		teacherRepo:            teacherRepo,
	}
}

func (s *CourseService) CreateCourse(ctx context.Context, schoolID, name, description string) (*domain.Course, error) {
	course := &domain.Course{
		ID:          uuid.New().String(),
		SchoolID:    schoolID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.courseRepo.Create(ctx, course); err != nil {
		return nil, err
	}
	return course, nil
}

func (s *CourseService) ListCourses(ctx context.Context, schoolID string, page, size int) ([]domain.Course, int64, error) {
	return s.courseRepo.List(ctx, schoolID, page, size)
}

func (s *CourseService) ListCourseAssignments(ctx context.Context, schoolID, departmentID, classID string, page, size int) ([]domain.CourseAssignmentInfo, int64, error) {
	return s.courseRepo.ListAssignments(ctx, schoolID, departmentID, classID, page, size)
}

func (s *CourseService) UpdateCourse(ctx context.Context, id, name, description string) error {
	course, err := s.courseRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	course.Name = name
	course.Description = description
	course.UpdatedAt = time.Now()
	return s.courseRepo.Update(ctx, course)
}

func (s *CourseService) DeleteCourse(ctx context.Context, id string) error {
	return s.courseRepo.Delete(ctx, id)
}

func (s *CourseService) AssignCourse(ctx context.Context, schoolID, courseID, teacherID, classID string) (*domain.TeachingAssignment, error) {
	// Validate Teacher
	var tid *string
	if teacherID != "" {
		if _, err := s.teacherRepo.GetByID(ctx, teacherID); err != nil {
			return nil, errors.New("teacher_not_found")
		}
		tid = &teacherID
	}

	// Validate Course
	if _, err := s.courseRepo.GetByID(ctx, courseID); err != nil {
		return nil, errors.New("course_not_found")
	}

	// Check if assignment already exists
	existing, count, err := s.teachingAssignmentRepo.List(ctx, schoolID, courseID, teacherID, classID, 1, 1)
	if err == nil && count > 0 && len(existing) > 0 {
		return &existing[0], nil
	}

	assignment := &domain.TeachingAssignment{
		ID:        uuid.New().String(),
		SchoolID:  schoolID,
		CourseID:  courseID,
		TeacherID: tid,
		ClassID:   classID,
		CreatedAt: time.Now(),
	}
	if err := s.teachingAssignmentRepo.Create(ctx, assignment); err != nil {
		return nil, err
	}
	return assignment, nil
}

func (s *CourseService) BatchAssignCourse(ctx context.Context, schoolID, courseID, teacherID string, classIDs []string) error {
	if len(classIDs) == 0 {
		return nil
	}

	var tid *string
	if teacherID != "" {
		// Optional: Validate teacher existence here too if strictness is required
		// But for batch, maybe we trust the caller or let DB fail if ID is invalid but not empty
		tid = &teacherID
	}

	assignments := make([]domain.TeachingAssignment, len(classIDs))
	now := time.Now()
	for i, classID := range classIDs {
		assignments[i] = domain.TeachingAssignment{
			ID:        uuid.New().String(),
			SchoolID:  schoolID,
			CourseID:  courseID,
			TeacherID: tid,
			ClassID:   classID,
			CreatedAt: now,
		}
	}
	return s.teachingAssignmentRepo.BatchCreate(ctx, assignments)
}

func (s *CourseService) ListAssignments(ctx context.Context, schoolID, courseID, teacherID, classID string, page, size int) ([]domain.TeachingAssignmentDetail, int64, error) {
	return s.teachingAssignmentRepo.ListDetails(ctx, schoolID, courseID, teacherID, classID, page, size)
}

func (s *CourseService) UpdateAssignment(ctx context.Context, id, teacherID, classID string) error {
	assignment, err := s.teachingAssignmentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate Teacher
	var tid *string
	if teacherID != "" {
		if _, err := s.teacherRepo.GetByID(ctx, teacherID); err != nil {
			return errors.New("teacher_not_found")
		}
		tid = &teacherID
	}

	assignment.TeacherID = tid
	assignment.ClassID = classID
	// We might want to validate ClassID too, but let's assume it's valid or DB will error.

	return s.teachingAssignmentRepo.Update(ctx, assignment)
}

func (s *CourseService) RemoveAssignment(ctx context.Context, id string) error {
	return s.teachingAssignmentRepo.Delete(ctx, id)
}

func (s *CourseService) BatchRemoveAssignments(ctx context.Context, ids []string) error {
	return s.teachingAssignmentRepo.BatchDelete(ctx, ids)
}

func (s *CourseService) assignStudents(ctx context.Context, courseID string, studentIDs []string) error {
	if len(studentIDs) == 0 {
		return nil
	}
	enrollments := make([]domain.CourseStudent, len(studentIDs))
	now := time.Now()
	for i, studentID := range studentIDs {
		enrollments[i] = domain.CourseStudent{
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
