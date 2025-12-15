package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

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

var (
	// ErrAdminAccountNotFound indicates the managed account does not exist in the given school.
	ErrAdminAccountNotFound = errors.New("account not found")
	// ErrAdminAccountRoleNotSupported indicates the account is not a teacher or student.
	ErrAdminAccountRoleNotSupported = errors.New("account role not supported")
	// ErrAdminAccountAlreadyLocked indicates the account is already locked.
	ErrAdminAccountAlreadyLocked = errors.New("account already locked")
	// ErrAdminAccountNotLocked indicates the account is not in locked state when unlock is requested.
	ErrAdminAccountNotLocked = errors.New("account is not locked")
	// ErrAdminPasswordResetPending indicates a reset request is already pending.
	ErrAdminPasswordResetPending = errors.New("password reset already pending")
	// ErrAdminBatchAccountIDsRequired indicates batch operations lack targets.
	ErrAdminBatchAccountIDsRequired = errors.New("account_ids required")
	// ErrAdminBatchActionUnsupported indicates action field invalid.
	ErrAdminBatchActionUnsupported = errors.New("unsupported batch action")
)

// AdminBatchAction enumerates supported administrative bulk operations.
type AdminBatchAction string

const (
	AdminBatchActionLock          AdminBatchAction = "lock"
	AdminBatchActionUnlock        AdminBatchAction = "unlock"
	AdminBatchActionResetPassword AdminBatchAction = "reset_password"
	AdminBatchActionDelete        AdminBatchAction = "delete"
)

var adminBatchActionSet = map[AdminBatchAction]struct{}{
	AdminBatchActionLock:          {},
	AdminBatchActionUnlock:        {},
	AdminBatchActionResetPassword: {},
	AdminBatchActionDelete:        {},
}

// AdminBatchOperationInput wraps request parameters for batch account operations.
type AdminBatchOperationInput struct {
	SchoolID   string
	AccountIDs []string
	Action     AdminBatchAction
}

// AdminBatchOperationResult summarizes execution outcome of a batch action.
type AdminBatchOperationResult struct {
	Succeeded []string          `json:"succeeded"`
	Failed    map[string]string `json:"failed"`
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
		Status:       domain.AccountStatusActive,
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
		Status:       domain.AccountStatusActive,
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

// AccountDepartmentScope controls department-based filtering.
type AccountDepartmentScope string

const (
	// AccountDepartmentScopeAll keeps all accounts regardless of department state.
	AccountDepartmentScopeAll AccountDepartmentScope = "all"
	// AccountDepartmentScopeUnassigned keeps accounts without department association.
	AccountDepartmentScopeUnassigned AccountDepartmentScope = "unassigned"
)

// AccountClassScope controls class-based filtering.
type AccountClassScope string

const (
	// AccountClassScopeAll keeps all accounts regardless of class state.
	AccountClassScopeAll AccountClassScope = "all"
	// AccountClassScopeUnassigned keeps accounts without class association.
	AccountClassScopeUnassigned AccountClassScope = "unassigned"
)

// ListAccountsOptions configures account listing behaviour.
type ListAccountsOptions struct {
	SchoolID        string
	Role            domain.Role
	Status          domain.AccountStatus
	DepartmentID    string
	DepartmentScope AccountDepartmentScope
	ClassID         string
	ClassScope      AccountClassScope
	CourseID        string
	Page            int
	Size            int
	Query           string
}

// AdminAccountSummary represents account information for admin UI.
type AdminAccountSummary struct {
	ID           string      `json:"id"`
	Role         domain.Role `json:"role"`
	Identifier   string      `json:"identifier"`
	Name         string      `json:"name"`
	Email        string      `json:"email,omitempty"`
	Phone        string      `json:"phone,omitempty"`
	DepartmentID string      `json:"department_id,omitempty"`
	Department   string      `json:"department,omitempty"`
	ClassID      string      `json:"class_id,omitempty"`
	ClassName    string      `json:"class_name,omitempty"`
	Status       string      `json:"status"`
	LastActiveAt *time.Time  `json:"last_active_at,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

// ListAccounts returns accounts filtered by role for an administrator.
func (s *AdminService) ListAccounts(ctx context.Context, opts ListAccountsOptions) ([]AdminAccountSummary, int64, error) {
	if opts.SchoolID == "" {
		return nil, 0, errors.New("school_id required")
	}
	if opts.Role != "" && opts.Role != domain.RoleTeacher && opts.Role != domain.RoleStudent {
		return nil, 0, errors.New("unsupported role")
	}

	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Size <= 0 {
		opts.Size = 50
	}
	if opts.Size > 200 {
		opts.Size = 200
	}

	onlyClassless := opts.ClassScope == AccountClassScopeUnassigned
	onlyDepartmentless := opts.DepartmentScope == AccountDepartmentScopeUnassigned
	accounts, total, err := s.accounts.ListByRole(
		ctx,
		opts.SchoolID,
		opts.Role,
		opts.Status,
		opts.DepartmentID,
		opts.ClassID,
		opts.CourseID,
		onlyClassless,
		onlyDepartmentless,
		opts.Page,
		opts.Size,
		opts.Query,
	)
	if err != nil {
		return nil, 0, err
	}

	summaries := make([]AdminAccountSummary, 0, len(accounts))
	for _, account := range accounts {
		summary := AdminAccountSummary{
			ID:         account.ID,
			Role:       account.Role,
			Identifier: account.Identifier,
			Name:       account.DisplayName,
			Status:     string(account.Status),
			CreatedAt:  account.CreatedAt,
		}

		if lastActive := account.UpdatedAt; !lastActive.IsZero() {
			v := lastActive
			summary.LastActiveAt = &v
		}

		switch account.Role {
		case domain.RoleTeacher:
			profile, perr := s.teachers.GetByAccountID(ctx, account.ID)
			if perr != nil {
				return nil, 0, perr
			}
			if profile != nil {
				summary.Email = profile.Email
				summary.Phone = profile.Phone
				if profile.DepartmentID != nil {
					summary.DepartmentID = *profile.DepartmentID
					department, derr := s.departments.GetByID(ctx, *profile.DepartmentID)
					if derr == nil && department != nil {
						summary.Department = department.Name
					}
				}
			}
		case domain.RoleStudent:
			profile, perr := s.students.GetByAccountID(ctx, account.ID)
			if perr != nil {
				return nil, 0, perr
			}
			if profile != nil {
				summary.Email = profile.Email
				summary.Phone = profile.Phone
				summary.ClassID = profile.ClassID

				class, cerr := s.classes.GetByID(ctx, profile.ClassID)
				if cerr == nil && class != nil {
					summary.ClassName = class.Name
					summary.DepartmentID = class.DepartmentID
					department, derr := s.departments.GetByID(ctx, class.DepartmentID)
					if derr == nil && department != nil {
						summary.Department = department.Name
					}
				}
			}
		default:
			// Skip roles that are not managed in account center.
			continue
		}

		if opts.ClassScope == AccountClassScopeUnassigned && summary.ClassID != "" {
			continue
		}
		if opts.ClassID != "" && summary.ClassID != opts.ClassID {
			continue
		}
		if opts.DepartmentScope == AccountDepartmentScopeUnassigned && summary.DepartmentID != "" {
			continue
		}
		if opts.DepartmentID != "" && summary.DepartmentID != opts.DepartmentID {
			continue
		}

		summaries = append(summaries, summary)
	}

	var filteredTotal int64 = total
	if opts.ClassID != "" || opts.DepartmentID != "" || opts.DepartmentScope == AccountDepartmentScopeUnassigned || opts.ClassScope == AccountClassScopeUnassigned {
		filteredTotal = int64(len(summaries))
	}

	return summaries, filteredTotal, nil
}

// BatchOperateAccounts executes a single administrative action across multiple accounts.
func (s *AdminService) BatchOperateAccounts(ctx context.Context, input AdminBatchOperationInput) (*AdminBatchOperationResult, error) {
	if input.SchoolID == "" {
		return nil, errors.New("school_id required")
	}
	ids := deduplicateStrings(input.AccountIDs)
	if len(ids) == 0 {
		return nil, ErrAdminBatchAccountIDsRequired
	}
	if _, ok := adminBatchActionSet[input.Action]; !ok {
		return nil, ErrAdminBatchActionUnsupported
	}

	result := &AdminBatchOperationResult{
		Succeeded: make([]string, 0, len(ids)),
		Failed:    make(map[string]string),
	}

	for _, accountID := range ids {
		var err error
		switch input.Action {
		case AdminBatchActionLock:
			err = s.LockAccount(ctx, input.SchoolID, accountID)
		case AdminBatchActionUnlock:
			err = s.UnlockAccount(ctx, input.SchoolID, accountID)
		case AdminBatchActionResetPassword:
			err = s.ResetAccountPassword(ctx, input.SchoolID, accountID)
		case AdminBatchActionDelete:
			err = s.DeleteAccount(ctx, input.SchoolID, accountID)
		default:
			err = ErrAdminBatchActionUnsupported
		}

		if err != nil {
			result.Failed[accountID] = err.Error()
			continue
		}
		result.Succeeded = append(result.Succeeded, accountID)
	}

	return result, nil
}

func (s *AdminService) ResetAccountPassword(ctx context.Context, schoolID, accountID string) error {
	account, err := s.ensureManageableAccount(ctx, schoolID, accountID)
	if err != nil {
		return err
	}

	if account.Status == domain.AccountStatusPasswordResetRequired {
		return ErrAdminPasswordResetPending
	}

	if err := s.accounts.UpdateStatus(ctx, accountID, schoolID, domain.AccountStatusPasswordResetRequired); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminAccountNotFound
		}
		return err
	}
	return nil
}

func (s *AdminService) LockAccount(ctx context.Context, schoolID, accountID string) error {
	account, err := s.ensureManageableAccount(ctx, schoolID, accountID)
	if err != nil {
		return err
	}

	if account.Status == domain.AccountStatusLocked {
		return ErrAdminAccountAlreadyLocked
	}

	if err := s.accounts.UpdateStatus(ctx, accountID, schoolID, domain.AccountStatusLocked); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminAccountNotFound
		}
		return err
	}
	return nil
}

func (s *AdminService) UnlockAccount(ctx context.Context, schoolID, accountID string) error {
	account, err := s.ensureManageableAccount(ctx, schoolID, accountID)
	if err != nil {
		return err
	}

	if account.Status != domain.AccountStatusLocked {
		return ErrAdminAccountNotLocked
	}

	if err := s.accounts.UpdateStatus(ctx, accountID, schoolID, domain.AccountStatusActive); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminAccountNotFound
		}
		return err
	}
	return nil
}

func (s *AdminService) DeleteAccount(ctx context.Context, schoolID, accountID string) error {
	if _, err := s.ensureManageableAccount(ctx, schoolID, accountID); err != nil {
		return err
	}

	if err := s.accounts.Delete(ctx, accountID, schoolID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminAccountNotFound
		}
		return err
	}
	return nil
}

func (s *AdminService) ensureManageableAccount(ctx context.Context, schoolID, accountID string) (*domain.Account, error) {
	if accountID == "" {
		return nil, ErrAdminAccountNotFound
	}
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminAccountNotFound
		}
		return nil, err
	}
	if account == nil || account.SchoolID != schoolID {
		return nil, ErrAdminAccountNotFound
	}
	if account.Role != domain.RoleTeacher && account.Role != domain.RoleStudent {
		return nil, ErrAdminAccountRoleNotSupported
	}
	return account, nil
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

func deduplicateStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	clean := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		clean = append(clean, trimmed)
	}
	return clean
}

// UpdateAccountStructureInput defines parameters for updating account belonging.
type UpdateAccountStructureInput struct {
	SchoolID     string
	AccountID    string
	DepartmentID *string
	ClassID      *string
}

// UpdateAccountStructure updates the belonging structure (class/department) of an account.
func (s *AdminService) UpdateAccountStructure(ctx context.Context, input UpdateAccountStructureInput) error {
	account, err := s.accounts.FindByID(ctx, input.AccountID)
	if err != nil {
		return err
	}
	if account.SchoolID != input.SchoolID {
		return ErrAdminAccountNotFound
	}

	if account.Role == domain.RoleStudent {
		if input.ClassID == nil {
			return nil
		}
		student, err := s.students.GetByAccountID(ctx, input.AccountID)
		if err != nil {
			return err
		}
		if student == nil {
			return errors.New("student profile not found")
		}

		classID := strings.TrimSpace(*input.ClassID)
		if classID != "" {
			class, err := s.classes.GetByID(ctx, classID)
			if err != nil {
				return err
			}
			if class.SchoolID != input.SchoolID {
				return errors.New("class not found in school")
			}
		}

		return s.students.UpdateClassID(ctx, student.ID, classID)
	} else if account.Role == domain.RoleTeacher {
		teacher, err := s.teachers.GetByAccountID(ctx, input.AccountID)
		if err != nil {
			return err
		}
		if teacher == nil {
			return errors.New("teacher profile not found")
		}

		var deptID *string
		if input.DepartmentID != nil {
			val := strings.TrimSpace(*input.DepartmentID)
			if val != "" {
				dept, err := s.departments.GetByID(ctx, val)
				if err != nil {
					return err
				}
				if dept.SchoolID != input.SchoolID {
					return errors.New("department not found in school")
				}
				deptID = &val
			}
		}

		return s.teachers.UpdateDepartmentID(ctx, teacher.ID, deptID)
	}

	return nil
}
