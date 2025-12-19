package service

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"github.com/google/uuid"
)

type CourseService struct {
	courseRepo        repository.CourseRepository
	courseStudentRepo repository.CourseStudentRepository
	studentRepo       repository.StudentRepository
	teacherRepo       repository.TeacherRepository
}

func NewCourseService(
	courseRepo repository.CourseRepository,
	courseStudentRepo repository.CourseStudentRepository,
	studentRepo repository.StudentRepository,
	teacherRepo repository.TeacherRepository,
) *CourseService {
	return &CourseService{
		courseRepo:        courseRepo,
		courseStudentRepo: courseStudentRepo,
		studentRepo:       studentRepo,
		teacherRepo:       teacherRepo,
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
