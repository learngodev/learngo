package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
	"learn-go/pkg/crypto"
)

// AdminService manages administrative operations.
type AdminService struct {
	accounts     repository.AccountRepository
	teachers     repository.TeacherRepository
	students     repository.StudentRepository
	departments  repository.DepartmentRepository
	classes      repository.ClassRepository
	teacherLinks repository.TeacherStudentRepository
}

// NewAdminService constructs an AdminService.
func NewAdminService(acc repository.AccountRepository, teachers repository.TeacherRepository, students repository.StudentRepository, departments repository.DepartmentRepository, classes repository.ClassRepository, links repository.TeacherStudentRepository) *AdminService {
	return &AdminService{
		accounts:     acc,
		teachers:     teachers,
		students:     students,
		departments:  departments,
		classes:      classes,
		teacherLinks: links,
	}
}

// CreateTeacher creates a teacher account with default password.
type CreateTeacherInput struct {
	SchoolID   string
	Number     string
	Name       string
	Email      string
	Phone      string
	DefaultPwd string
}

func (s *AdminService) CreateTeacher(ctx context.Context, input CreateTeacherInput) (*domain.Teacher, error) {
	if input.DefaultPwd == "" {
		return nil, errors.New("default password required")
	}

	hash, err := crypto.HashPassword(input.DefaultPwd)
	if err != nil {
		return nil, err
	}

	account := &domain.Account{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		Role:         domain.RoleTeacher,
		Identifier:   input.Number,
		PasswordHash: hash,
		DisplayName:  input.Name,
	}

	if err := s.accounts.Create(ctx, account); err != nil {
		return nil, err
	}

	teacher := &domain.Teacher{
		ID:        uuid.NewString(),
		SchoolID:  input.SchoolID,
		AccountID: account.ID,
		Number:    input.Number,
		Email:     input.Email,
		Phone:     input.Phone,
	}

	if err := s.teachers.Create(ctx, teacher); err != nil {
		return nil, err
	}

	return teacher, nil
}

// CreateStudentInput contains data for student creation.
type CreateStudentInput struct {
	SchoolID   string
	Number     string
	Name       string
	Email      string
	Phone      string
	ClassID    string
	DefaultPwd string
	TeacherIDs []string
}

func (s *AdminService) CreateStudent(ctx context.Context, input CreateStudentInput) (*domain.Student, error) {
	if input.DefaultPwd == "" {
		return nil, errors.New("default password required")
	}
	if len(input.TeacherIDs) == 0 {
		return nil, errors.New("at least one teacher required")
	}

	hash, err := crypto.HashPassword(input.DefaultPwd)
	if err != nil {
		return nil, err
	}

	account := &domain.Account{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		Role:         domain.RoleStudent,
		Identifier:   input.Number,
		PasswordHash: hash,
		DisplayName:  input.Name,
	}

	if err := s.accounts.Create(ctx, account); err != nil {
		return nil, err
	}

	student := &domain.Student{
		ID:        uuid.NewString(),
		SchoolID:  input.SchoolID,
		AccountID: account.ID,
		Number:    input.Number,
		ClassID:   input.ClassID,
		Email:     input.Email,
		Phone:     input.Phone,
	}

	if err := s.students.Create(ctx, student); err != nil {
		return nil, err
	}

	if err := s.teacherLinks.BindTeachers(ctx, student.ID, input.TeacherIDs); err != nil {
		return nil, err
	}

	return student, nil
}

// CreateDepartment registers a new department.
func (s *AdminService) CreateDepartment(ctx context.Context, schoolID, name string) (*domain.Department, error) {
	department := &domain.Department{
		ID:       uuid.NewString(),
		SchoolID: schoolID,
		Name:     name,
	}
	if err := s.departments.Create(ctx, department); err != nil {
		return nil, err
	}
	return department, nil
}

// CreateClass registers a class under department.
func (s *AdminService) CreateClass(ctx context.Context, schoolID, departmentID, name string) (*domain.Class, error) {
	class := &domain.Class{
		ID:           uuid.NewString(),
		SchoolID:     schoolID,
		DepartmentID: departmentID,
		Name:         name,
	}
	if err := s.classes.Create(ctx, class); err != nil {
		return nil, err
	}
	return class, nil
}

// ListDepartments returns departments under a school.
func (s *AdminService) ListDepartments(ctx context.Context, schoolID string) ([]domain.Department, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}
	return s.departments.List(ctx, schoolID)
}

// ListClasses returns classes optionally filtered by department.
func (s *AdminService) ListClasses(ctx context.Context, schoolID, departmentID string) ([]domain.Class, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}
	if departmentID == "" {
		return nil, errors.New("department_id required")
	}
	return s.classes.ListByDepartment(ctx, schoolID, departmentID)
}

// UpdateDepartment renames a department within a school.
func (s *AdminService) UpdateDepartment(ctx context.Context, schoolID, departmentID, name string) (*domain.Department, error) {
	trimmed := strings.TrimSpace(name)
	if schoolID == "" || departmentID == "" {
		return nil, errors.New("school_id and department_id required")
	}
	if trimmed == "" {
		return nil, errors.New("department name required")
	}

	department, err := s.departments.GetByID(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	if department.SchoolID != schoolID {
		return nil, errors.New("department does not belong to school")
	}

	if err := s.departments.UpdateName(ctx, departmentID, schoolID, trimmed); err != nil {
		return nil, err
	}
	department.Name = trimmed
	return department, nil
}

// DeleteDepartment removes a department if it has no classes.
func (s *AdminService) DeleteDepartment(ctx context.Context, schoolID, departmentID string) error {
	if schoolID == "" || departmentID == "" {
		return errors.New("school_id and department_id required")
	}

	department, err := s.departments.GetByID(ctx, departmentID)
	if err != nil {
		return err
	}
	if department.SchoolID != schoolID {
		return errors.New("department does not belong to school")
	}

	classes, err := s.classes.ListByDepartment(ctx, schoolID, departmentID)
	if err != nil {
		return err
	}
	if len(classes) > 0 {
		return errors.New("department contains classes")
	}

	return s.departments.Delete(ctx, departmentID, schoolID)
}

// UpdateClass renames a class.
func (s *AdminService) UpdateClass(ctx context.Context, schoolID, classID, name string) (*domain.Class, error) {
	trimmed := strings.TrimSpace(name)
	if schoolID == "" || classID == "" {
		return nil, errors.New("school_id and class_id required")
	}
	if trimmed == "" {
		return nil, errors.New("class name required")
	}

	class, err := s.classes.GetByID(ctx, classID)
	if err != nil {
		return nil, err
	}
	if class.SchoolID != schoolID {
		return nil, errors.New("class does not belong to school")
	}

	if err := s.classes.UpdateName(ctx, classID, schoolID, trimmed); err != nil {
		return nil, err
	}
	class.Name = trimmed
	return class, nil
}

// DeleteClass removes a class record.
func (s *AdminService) DeleteClass(ctx context.Context, schoolID, classID string) error {
	if schoolID == "" || classID == "" {
		return errors.New("school_id and class_id required")
	}

	class, err := s.classes.GetByID(ctx, classID)
	if err != nil {
		return err
	}
	if class.SchoolID != schoolID {
		return errors.New("class does not belong to school")
	}

	return s.classes.Delete(ctx, classID, schoolID)
}
