package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"learn-go/internal/domain"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain/aiagent"
	"learn-go/internal/domain/identity"
	"learn-go/internal/domain/organization"
	"learn-go/internal/domain/roster"
	"learn-go/internal/domain/scheduling"
)

// AdminUsecase 负责管理端相关操作。
type AdminUsecase struct {
	accounts     identity.AccountRepository
	teachers     roster.TeacherRepository
	students     roster.StudentRepository
	departments  organization.DepartmentRepository
	classes      organization.ClassRepository
	schedules    scheduling.CourseScheduleRepository
	teacherLinks roster.TeacherStudentRepository
	aiModel      AIChatModel
	aiSettings   aiagent.AIAgentSettingRepository
	hasher       identity.PasswordHasher
}

var (
	// ErrAdminAccountNotFound 表示目标账号在该学校中不存在。
	ErrAdminAccountNotFound = errors.New("account not found")
	// ErrAdminAccountRoleNotSupported 表示该账号既不是教师也不是学生。
	ErrAdminAccountRoleNotSupported = errors.New("account role not supported")
	// ErrAdminAccountAlreadyLocked 表示账号已被锁定。
	ErrAdminAccountAlreadyLocked = errors.New("account already locked")
	// ErrAdminAccountNotLocked 表示请求解锁时账号并未处于锁定状态。
	ErrAdminAccountNotLocked = errors.New("account is not locked")
	// ErrAdminPasswordResetPending 表示已有待处理的密码重置请求。
	ErrAdminPasswordResetPending = errors.New("password reset already pending")
	// ErrAdminBatchAccountIDsRequired 表示批量操作缺少目标账号。
	ErrAdminBatchAccountIDsRequired = errors.New("account_ids required")
	// ErrAdminBatchActionUnsupported 表示 action 字段无效。
	ErrAdminBatchActionUnsupported = errors.New("unsupported batch action")
)

// AdminBatchAction 枚举支持的管理批量操作。
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

// AdminBatchOperationInput 封装账号批量操作的请求参数。
type AdminBatchOperationInput struct {
	SchoolID   string
	AccountIDs []string
	Action     AdminBatchAction
}

// AdminBatchOperationResult 汇总批量操作的执行结果。
type AdminBatchOperationResult struct {
	Succeeded []string          `json:"succeeded"`
	Failed    map[string]string `json:"failed"`
}

// AIOperation 表示从 AI 指令中解析出的单个操作。
type AIOperation struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// AIAnalyzeResponse 表示 AI 的分析结果。
type AIAnalyzeResponse struct {
	Operations []AIOperation `json:"operations"`
	Analysis   string        `json:"analysis"`
}

// NewAdminUsecase 构造 AdminUsecase。
func NewAdminUsecase(
	acc identity.AccountRepository,
	teachers roster.TeacherRepository,
	students roster.StudentRepository,
	departments organization.DepartmentRepository,
	classes organization.ClassRepository,
	schedules scheduling.CourseScheduleRepository,
	links roster.TeacherStudentRepository,
	aiModel AIChatModel,
	aiSettings aiagent.AIAgentSettingRepository,
	hasher identity.PasswordHasher,
) *AdminUsecase {
	return &AdminUsecase{
		accounts:     acc,
		teachers:     teachers,
		students:     students,
		departments:  departments,
		classes:      classes,
		schedules:    schedules,
		teacherLinks: links,
		aiModel:      aiModel,
		aiSettings:   aiSettings,
		hasher:       hasher,
	}
}

// CreateTeacher 使用默认密码创建教师账号。
type CreateTeacherInput struct {
	SchoolID   string
	Number     string
	Name       string
	Email      string
	Phone      string
	DefaultPwd string
}

func (s *AdminUsecase) CreateTeacher(ctx context.Context, input CreateTeacherInput) (*roster.Teacher, error) {
	if input.DefaultPwd == "" {
		return nil, errors.New("default password required")
	}

	hash, err := s.hasher.Hash(input.DefaultPwd)
	if err != nil {
		return nil, err
	}

	account := &identity.Account{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		Role:         identity.RoleTeacher,
		Status:       identity.AccountStatusActive,
		Identifier:   input.Number,
		PasswordHash: hash,
		DisplayName:  input.Name,
	}

	if err := s.accounts.Create(ctx, account); err != nil {
		return nil, err
	}

	var email *string
	if input.Email != "" {
		email = &input.Email
	}

	teacher := &roster.Teacher{
		ID:        uuid.NewString(),
		SchoolID:  input.SchoolID,
		AccountID: account.ID,
		Number:    input.Number,
		Email:     email,
		Phone:     input.Phone,
	}

	if err := s.teachers.Create(ctx, teacher); err != nil {
		return nil, err
	}

	return teacher, nil
}

// CreateStudentInput 包含创建学生所需的数据。
type CreateStudentInput struct {
	SchoolID   string
	Number     string
	Name       string
	Email      string
	Phone      string
	ClassID    string
	DefaultPwd string
}

func (s *AdminUsecase) CreateStudent(ctx context.Context, input CreateStudentInput) (*roster.Student, error) {
	if input.DefaultPwd == "" {
		return nil, errors.New("default password required")
	}
	if strings.TrimSpace(input.SchoolID) == "" {
		return nil, errors.New("school_id required")
	}
	if strings.TrimSpace(input.ClassID) != "" {
		class, err := s.classes.GetByID(ctx, input.ClassID)
		if err != nil {
			return nil, err
		}
		if class == nil || class.SchoolID != input.SchoolID {
			return nil, errors.New("class does not belong to school")
		}
	}

	hash, err := s.hasher.Hash(input.DefaultPwd)
	if err != nil {
		return nil, err
	}

	account := &identity.Account{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		Role:         identity.RoleStudent,
		Status:       identity.AccountStatusActive,
		Identifier:   input.Number,
		PasswordHash: hash,
		DisplayName:  input.Name,
	}

	if err := s.accounts.Create(ctx, account); err != nil {
		return nil, err
	}

	var email *string
	if input.Email != "" {
		email = &input.Email
	}

	student := &roster.Student{
		ID:        uuid.NewString(),
		SchoolID:  input.SchoolID,
		AccountID: account.ID,
		Number:    input.Number,
		Email:     email,
		Phone:     input.Phone,
	}

	if input.ClassID != "" {
		student.ClassID = &input.ClassID
	}

	if err := s.students.Create(ctx, student); err != nil {
		return nil, err
	}

	return student, nil
}

// AccountDepartmentScope 控制按院系的筛选方式。
type AccountDepartmentScope string

const (
	// AccountDepartmentScopeAll 保留所有账号，不考虑院系归属。
	AccountDepartmentScopeAll AccountDepartmentScope = "all"
	// AccountDepartmentScopeUnassigned 仅保留未关联院系的账号。
	AccountDepartmentScopeUnassigned AccountDepartmentScope = "unassigned"
)

// AccountClassScope 控制按班级的筛选方式。
type AccountClassScope string

const (
	// AccountClassScopeAll 保留所有账号，不考虑班级归属。
	AccountClassScopeAll AccountClassScope = "all"
	// AccountClassScopeUnassigned 仅保留未关联班级的账号。
	AccountClassScopeUnassigned AccountClassScope = "unassigned"
)

// ListAccountsOptions 配置账号列表的查询行为。
type ListAccountsOptions struct {
	SchoolID        string
	Role            identity.Role
	Status          identity.AccountStatus
	DepartmentID    string
	DepartmentScope AccountDepartmentScope
	ClassID         string
	ClassScope      AccountClassScope
	CourseID        string
	Page            int
	Size            int
	Query           string
}

// AdminAccountSummary 表示管理端界面展示的账号信息。
type AdminAccountSummary struct {
	ID           string        `json:"id"`
	ProfileID    string        `json:"profile_id,omitempty"`
	Role         identity.Role `json:"role"`
	Identifier   string        `json:"identifier"`
	Name         string        `json:"name"`
	Email        *string       `json:"email,omitempty"`
	Phone        string        `json:"phone,omitempty"`
	DepartmentID []string      `json:"department_id,omitempty"`
	Department   []string      `json:"department,omitempty"`
	ClassID      []string      `json:"class_id,omitempty"`
	ClassName    []string      `json:"class_name,omitempty"`
	Status       string        `json:"status"`
	LastActiveAt *time.Time    `json:"last_active_at,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
}

// ListAccounts 为管理员返回按角色筛选的账号。
func (s *AdminUsecase) ListAccounts(ctx context.Context, opts ListAccountsOptions) ([]AdminAccountSummary, int64, error) {
	if opts.SchoolID == "" {
		return nil, 0, errors.New("school_id required")
	}
	if opts.Role != "" && opts.Role != identity.RoleTeacher && opts.Role != identity.RoleStudent {
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
		case identity.RoleTeacher:
			profile, perr := s.teachers.GetByAccountID(ctx, account.ID)
			if perr != nil {
				return nil, 0, perr
			}
			if profile != nil {
				summary.ProfileID = profile.ID
				summary.Email = profile.Email
				summary.Phone = profile.Phone

				classIDs := make(map[string]struct{})
				departmentIDs := make(map[string]struct{})
				classNames := make(map[string]struct{})
				departmentNames := make(map[string]struct{})

				if profile.DepartmentID != nil && *profile.DepartmentID != "" {
					departmentIDs[*profile.DepartmentID] = struct{}{}
				}

				if s.schedules != nil {
					schedules, serr := s.schedules.ListByTeacher(ctx, profile.ID)
					if serr != nil {
						return nil, 0, serr
					}

					for _, sch := range schedules {
						if sch.ClassID == "" {
							continue
						}
						classIDs[sch.ClassID] = struct{}{}
					}
				}

				for classID := range classIDs {
					class, cerr := s.classes.GetByID(ctx, classID)
					if cerr != nil || class == nil {
						continue
					}
					if class.Name != "" {
						classNames[class.Name] = struct{}{}
					}
					if class.DepartmentID != "" {
						departmentIDs[class.DepartmentID] = struct{}{}
					}
				}

				for departmentID := range departmentIDs {
					department, derr := s.departments.GetByID(ctx, departmentID)
					if derr != nil || department == nil || department.Name == "" {
						continue
					}
					departmentNames[department.Name] = struct{}{}
				}

				summary.ClassID = mapKeysSorted(classIDs)
				summary.ClassName = mapKeysSorted(classNames)
				summary.DepartmentID = mapKeysSorted(departmentIDs)
				summary.Department = mapKeysSorted(departmentNames)
			}
		case identity.RoleStudent:
			profile, perr := s.students.GetByAccountID(ctx, account.ID)
			if perr != nil {
				return nil, 0, perr
			}
			if profile != nil {
				summary.ProfileID = profile.ID
				summary.Email = profile.Email
				summary.Phone = profile.Phone
				if profile.ClassID != nil {
					summary.ClassID = []string{*profile.ClassID}

					class, cerr := s.classes.GetByID(ctx, *profile.ClassID)
					if cerr == nil && class != nil {
						summary.ClassName = []string{class.Name}
						summary.DepartmentID = []string{class.DepartmentID}
						department, derr := s.departments.GetByID(ctx, class.DepartmentID)
						if derr == nil && department != nil {
							summary.Department = []string{department.Name}
						}
					}
				}
			}
		default:
			// 跳过不在账号中心管理范围内的角色。
			continue
		}

		if opts.ClassScope == AccountClassScopeUnassigned && len(summary.ClassID) > 0 {
			// 防御性处理：查询未分班账号却返回了有班级的账号时跳过
			continue
		}
		// 仓储层已负责按 ClassID 筛选，包括学生（班级成员）
		// 和教师（排课分配）。这里不应再按 summary.ClassID 过滤，
		// 因为教师档案上没有 ClassID 属性。

		if opts.DepartmentScope == AccountDepartmentScopeUnassigned && len(summary.DepartmentID) > 0 {
			continue
		}
		// 同理，DepartmentID 的筛选也由仓储层处理。

		summaries = append(summaries, summary)
	}

	var filteredTotal int64 = total
	if opts.ClassID != "" || opts.DepartmentID != "" || opts.DepartmentScope == AccountDepartmentScopeUnassigned || opts.ClassScope == AccountClassScopeUnassigned {
		filteredTotal = int64(len(summaries))
	}

	return summaries, filteredTotal, nil
}

func mapKeysSorted(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	values := make([]string, 0, len(set))
	for value := range set {
		if strings.TrimSpace(value) == "" {
			continue
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return nil
	}
	sort.Strings(values)
	return values
}

// BatchOperateAccounts 对多个账号执行同一项管理操作。
func (s *AdminUsecase) BatchOperateAccounts(ctx context.Context, input AdminBatchOperationInput) (*AdminBatchOperationResult, error) {
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

func (s *AdminUsecase) ResetAccountPassword(ctx context.Context, schoolID, accountID string) error {
	account, err := s.ensureManageableAccount(ctx, schoolID, accountID)
	if err != nil {
		return err
	}

	if account.Status == identity.AccountStatusPasswordResetRequired {
		return ErrAdminPasswordResetPending
	}

	if err := s.accounts.UpdateStatus(ctx, accountID, schoolID, identity.AccountStatusPasswordResetRequired); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrAdminAccountNotFound
		}
		return err
	}
	return nil
}

func (s *AdminUsecase) LockAccount(ctx context.Context, schoolID, accountID string) error {
	account, err := s.ensureManageableAccount(ctx, schoolID, accountID)
	if err != nil {
		return err
	}

	if account.Status == identity.AccountStatusLocked {
		return ErrAdminAccountAlreadyLocked
	}

	if err := s.accounts.UpdateStatus(ctx, accountID, schoolID, identity.AccountStatusLocked); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrAdminAccountNotFound
		}
		return err
	}
	return nil
}

func (s *AdminUsecase) UnlockAccount(ctx context.Context, schoolID, accountID string) error {
	account, err := s.ensureManageableAccount(ctx, schoolID, accountID)
	if err != nil {
		return err
	}

	if account.Status != identity.AccountStatusLocked {
		return ErrAdminAccountNotLocked
	}

	if err := s.accounts.UpdateStatus(ctx, accountID, schoolID, identity.AccountStatusActive); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrAdminAccountNotFound
		}
		return err
	}
	return nil
}

func (s *AdminUsecase) DeleteAccount(ctx context.Context, schoolID, accountID string) error {
	if _, err := s.ensureManageableAccount(ctx, schoolID, accountID); err != nil {
		return err
	}

	if err := s.accounts.Delete(ctx, accountID, schoolID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrAdminAccountNotFound
		}
		return err
	}
	return nil
}

func (s *AdminUsecase) ensureManageableAccount(ctx context.Context, schoolID, accountID string) (*identity.Account, error) {
	if accountID == "" {
		return nil, ErrAdminAccountNotFound
	}
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrAdminAccountNotFound
		}
		return nil, err
	}
	if account == nil || account.SchoolID != schoolID {
		return nil, ErrAdminAccountNotFound
	}
	if account.Role != identity.RoleTeacher && account.Role != identity.RoleStudent {
		return nil, ErrAdminAccountRoleNotSupported
	}
	return account, nil
}

// CreateDepartment 新建院系。
func (s *AdminUsecase) CreateDepartment(ctx context.Context, schoolID, name string) (*organization.Department, error) {
	department := &organization.Department{
		ID:       uuid.NewString(),
		SchoolID: schoolID,
		Name:     name,
	}
	if err := s.departments.Create(ctx, department); err != nil {
		return nil, err
	}
	return department, nil
}

// CreateClass 在院系下新建班级。
func (s *AdminUsecase) CreateClass(ctx context.Context, schoolID, departmentID, name string) (*organization.Class, error) {
	if strings.TrimSpace(schoolID) == "" {
		return nil, errors.New("school_id required")
	}
	if strings.TrimSpace(departmentID) == "" {
		return nil, errors.New("department_id required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("class name required")
	}

	department, err := s.departments.GetByID(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	if department == nil || department.SchoolID != schoolID {
		return nil, errors.New("department does not belong to school")
	}

	class := &organization.Class{
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

// ListDepartments 返回某学校下的院系。
func (s *AdminUsecase) ListDepartments(ctx context.Context, schoolID string) ([]organization.Department, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}
	return s.departments.List(ctx, schoolID)
}

// ListClasses 返回班级列表，可按院系筛选。
func (s *AdminUsecase) ListClasses(ctx context.Context, schoolID, departmentID string) ([]organization.Class, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}
	if departmentID == "" {
		return nil, errors.New("department_id required")
	}
	return s.classes.ListByDepartment(ctx, schoolID, departmentID)
}

// UpdateDepartment 重命名学校内的院系。
func (s *AdminUsecase) UpdateDepartment(ctx context.Context, schoolID, departmentID, name string) (*organization.Department, error) {
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

// DeleteDepartment 在院系下没有班级时将其删除。
func (s *AdminUsecase) DeleteDepartment(ctx context.Context, schoolID, departmentID string) error {
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

// UpdateClass 重命名班级。
func (s *AdminUsecase) UpdateClass(ctx context.Context, schoolID, classID, name string) (*organization.Class, error) {
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

// DeleteClass 删除班级记录。
func (s *AdminUsecase) DeleteClass(ctx context.Context, schoolID, classID string) error {
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

// UpdateAccountInput 定义更新账号详情的参数。
type UpdateAccountInput struct {
	SchoolID  string
	AccountID string
	Name      *string
	Number    *string
	Email     *string
	Phone     *string
}

// UpdateAccount 更新账号基本信息。
func (s *AdminUsecase) UpdateAccount(ctx context.Context, input UpdateAccountInput) error {
	account, err := s.accounts.FindByID(ctx, input.AccountID)
	if err != nil {
		return err
	}
	if account.SchoolID != input.SchoolID {
		return ErrAdminAccountNotFound
	}

	// 更新 Account 字段
	updatedAccount := false
	if input.Name != nil {
		account.DisplayName = *input.Name
		updatedAccount = true
	}
	if input.Number != nil {
		account.Identifier = *input.Number
		updatedAccount = true
	}

	if updatedAccount {
		if err := s.accounts.Update(ctx, account); err != nil {
			return err
		}
	}

	// 更新角色特有字段
	if account.Role == identity.RoleStudent {
		student, err := s.students.GetByAccountID(ctx, input.AccountID)
		if err != nil {
			return err
		}
		if student != nil {
			updatedStudent := false
			if input.Number != nil {
				student.Number = *input.Number
				updatedStudent = true
			}
			if input.Email != nil {
				student.Email = input.Email
				updatedStudent = true
			}
			if input.Phone != nil {
				student.Phone = *input.Phone
				updatedStudent = true
			}
			if updatedStudent {
				if err := s.students.Update(ctx, student); err != nil {
					return err
				}
			}
		}
	} else if account.Role == identity.RoleTeacher {
		teacher, err := s.teachers.GetByAccountID(ctx, input.AccountID)
		if err != nil {
			return err
		}
		if teacher != nil {
			updatedTeacher := false
			if input.Number != nil {
				teacher.Number = *input.Number
				updatedTeacher = true
			}
			if input.Email != nil {
				teacher.Email = input.Email
				updatedTeacher = true
			}
			if input.Phone != nil {
				teacher.Phone = *input.Phone
				updatedTeacher = true
			}
			if updatedTeacher {
				if err := s.teachers.Update(ctx, teacher); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// UpdateAccountStructureInput 定义更新账号归属的参数。
type UpdateAccountStructureInput struct {
	SchoolID  string
	AccountID string
	ClassID   *string
}

// UpdateAccountStructure 更新账号的归属（班级/院系）。
func (s *AdminUsecase) UpdateAccountStructure(ctx context.Context, input UpdateAccountStructureInput) error {
	account, err := s.accounts.FindByID(ctx, input.AccountID)
	if err != nil {
		return err
	}
	if account.SchoolID != input.SchoolID {
		return ErrAdminAccountNotFound
	}

	if account.Role == identity.RoleStudent {
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
	}

	return nil
}

// AnalyzeBatchInstruction 分析自然语言指令并返回建议执行的操作。
func (s *AdminUsecase) AnalyzeBatchInstruction(ctx context.Context, schoolID uuid.UUID, instruction string) (*AIAnalyzeResponse, error) {
	setting, err := s.aiSettings.GetBySchoolID(ctx, schoolID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ai settings: %w", err)
	}
	if setting == nil {
		return nil, errors.New("no ai settings available")
	}

	setting.SystemPrompt = `You are an administrative assistant for a school management system.
Your task is to analyze the user's input and extract any valid administrative operations.
Ignore any conversational filler, greetings, or irrelevant text.

The supported operations are:
1. "create_student": {"name": "string", "email": "string" (optional), "password": "string", "number": "string", "department": "string" (optional), "class": "string" (optional)}
2. "create_teacher": {"name": "string", "email": "string" (optional), "password": "string", "number": "string"}
3. "lock_account": {"number": "string" (optional), "name": "string" (optional)}
4. "unlock_account": {"number": "string" (optional), "name": "string" (optional)}
5. "delete_account": {"number": "string" (optional), "name": "string" (optional)}
6. "update_account": {"number": "string" (optional), "name": "string" (optional), "email": "string" (optional), "phone": "string" (optional)}

Output MUST be a JSON object with two fields:
- "operations": An array of operation objects found in the input. Each object must have an "action" field and a "data" field. If no valid operations are found, this array should be empty.
- "analysis": A string field summarizing the actions to be taken in Chinese (Simplified). If no operations are found, explain why in Chinese.

Example 1 (Valid Command):
Input: "Create a student named John Doe in Computer Science class CS101 with password 123"
Output:
{
  "operations": [
    {"action": "create_student", "data": {"name": "John Doe", "password": "123", "number": "generated_or_placeholder", "department": "Computer Science", "class": "CS101"}}
  ],
  "analysis": "我将为 John Doe 创建一个学生账号，归属于计算机系 CS101 班。"
}

Example 2 (Mixed Input):
Input: "Hello there! Please lock account S12345. Thanks!"
Output:
{
  "operations": [
    {"action": "lock_account", "data": {"number": "S12345"}}
  ],
  "analysis": "我将锁定账号 S12345。"
}

Example 3 (Name Lookup):
Input: "Lock Zhang San's account"
Output:
{
  "operations": [
    {"action": "lock_account", "data": {"name": "Zhang San"}}
  ],
  "analysis": "我将锁定张三的账号。"
}

Example 4 (Irrelevant Input):
Input: "What is the weather today?"
Output:
{
  "operations": [],
  "analysis": "我只能协助处理学校管理任务，如创建账号或锁定用户。我无法回答关于天气的问题。"
}

Do not include any markdown formatting or explanation outside the JSON.`

	req := AIChatModelRequest{
		Setting: setting,
		Message: instruction,
	}

	resp, err := s.aiModel.GenerateResponse(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ai generation failed: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")

	var aiResp AIAnalyzeResponse

	if err := json.Unmarshal([]byte(content), &aiResp); err != nil {
		// 兼容旧格式或响应格式异常时的回退处理
		var ops []AIOperation
		if err2 := json.Unmarshal([]byte(content), &ops); err2 == nil {
			aiResp.Operations = ops
		} else {
			return nil, fmt.Errorf("failed to parse ai response: %w", err)
		}
	}

	return &aiResp, nil
}

// ExecuteBatchOperations 执行一组已确认的操作。
func (s *AdminUsecase) resolveAccount(ctx context.Context, schoolID, number, name string) (*identity.Account, error) {
	if number != "" {
		return s.accounts.FindByIdentifier(ctx, schoolID, number)
	}
	if name != "" {
		accounts, _, err := s.accounts.ListByRole(ctx, schoolID, "", "", "", "", "", false, false, 1, 2, name)
		if err != nil {
			return nil, err
		}
		if len(accounts) == 0 {
			return nil, fmt.Errorf("未找到名为 %s 的账号", name)
		}
		if len(accounts) > 1 {
			return nil, fmt.Errorf("找到多个名为 %s 的账号，请使用学号/工号", name)
		}
		return &accounts[0], nil
	}
	return nil, errors.New("需要提供学号/工号或姓名")
}

func getActionNameCN(action string) string {
	switch action {
	case "create_student":
		return "创建学生"
	case "create_teacher":
		return "创建教师"
	case "lock_account":
		return "锁定账号"
	case "unlock_account":
		return "解锁账号"
	case "delete_account":
		return "删除账号"
	case "update_account":
		return "更新账号"
	default:
		return action
	}
}

func (s *AdminUsecase) ExecuteBatchOperations(ctx context.Context, schoolID uuid.UUID, operations []AIOperation) ([]string, error) {
	var results []string
	for _, op := range operations {
		var res string
		var err error
		switch op.Action {
		case "create_student":
			var data struct {
				Name       string `json:"name"`
				Email      string `json:"email"`
				Password   string `json:"password"`
				Number     string `json:"number"`
				Department string `json:"department"`
				Class      string `json:"class"`
			}
			if err = json.Unmarshal(op.Data, &data); err == nil {
				number := data.Number
				if number == "generated_or_placeholder" || number == "" {
					number = "S" + uuid.NewString()[:8]
				}

				var classID string
				if data.Class != "" {
					classID, err = s.resolveClassID(ctx, schoolID.String(), data.Department, data.Class)
					if err != nil {
						results = append(results, fmt.Sprintf("Failed create_student %s: %v", data.Name, err))
						continue
					}
				}

				input := CreateStudentInput{
					SchoolID:   schoolID.String(),
					Name:       data.Name,
					Email:      data.Email,
					DefaultPwd: data.Password,
					Number:     number,
					ClassID:    classID,
				}
				_, err = s.CreateStudent(ctx, input)
				if err == nil {
					res = fmt.Sprintf("已创建学生 %s (%s)", data.Name, number)
				}
			}
		case "create_teacher":
			var data struct {
				Name     string `json:"name"`
				Email    string `json:"email"`
				Password string `json:"password"`
				Number   string `json:"number"`
			}
			if err = json.Unmarshal(op.Data, &data); err == nil {
				number := data.Number
				if number == "generated_or_placeholder" || number == "" {
					number = "T" + uuid.NewString()[:8]
				}
				input := CreateTeacherInput{
					SchoolID:   schoolID.String(),
					Name:       data.Name,
					Email:      data.Email,
					DefaultPwd: data.Password,
					Number:     number,
				}
				_, err = s.CreateTeacher(ctx, input)
				if err == nil {
					res = fmt.Sprintf("已创建教师 %s (%s)", data.Name, number)
				}
			}
		case "lock_account":
			var data struct {
				Number string `json:"number"`
				Name   string `json:"name"`
			}
			if err = json.Unmarshal(op.Data, &data); err == nil {
				acc, errFind := s.resolveAccount(ctx, schoolID.String(), data.Number, data.Name)
				if errFind == nil && acc != nil {
					err = s.LockAccount(ctx, schoolID.String(), acc.ID)
					if err == nil {
						res = fmt.Sprintf("已锁定账号 %s", acc.Identifier)
					}
				} else {
					err = errFind
				}
			}
		case "unlock_account":
			var data struct {
				Number string `json:"number"`
				Name   string `json:"name"`
			}
			if err = json.Unmarshal(op.Data, &data); err == nil {
				acc, errFind := s.resolveAccount(ctx, schoolID.String(), data.Number, data.Name)
				if errFind == nil && acc != nil {
					err = s.UnlockAccount(ctx, schoolID.String(), acc.ID)
					if err == nil {
						res = fmt.Sprintf("已解锁账号 %s", acc.Identifier)
					}
				} else {
					err = errFind
				}
			}
		case "delete_account":
			var data struct {
				Number string `json:"number"`
				Name   string `json:"name"`
			}
			if err = json.Unmarshal(op.Data, &data); err == nil {
				acc, errFind := s.resolveAccount(ctx, schoolID.String(), data.Number, data.Name)
				if errFind == nil && acc != nil {
					err = s.DeleteAccount(ctx, schoolID.String(), acc.ID)
					if err == nil {
						res = fmt.Sprintf("已删除账号 %s", acc.Identifier)
					}
				} else {
					err = errFind
				}
			}
		case "update_account":
			var data struct {
				Number string  `json:"number"`
				Name   *string `json:"name"`
				Email  *string `json:"email"`
				Phone  *string `json:"phone"`
			}
			if err = json.Unmarshal(op.Data, &data); err == nil {
				var lookupName string
				if data.Name != nil {
					lookupName = *data.Name
				}
				acc, errFind := s.resolveAccount(ctx, schoolID.String(), data.Number, lookupName)
				if errFind == nil && acc != nil {
					input := UpdateAccountInput{
						SchoolID:  schoolID.String(),
						AccountID: acc.ID,
						Name:      data.Name,
						Email:     data.Email,
						Phone:     data.Phone,
					}
					err = s.UpdateAccount(ctx, input)
					if err == nil {
						res = fmt.Sprintf("已更新账号 %s", acc.Identifier)
					}
				} else {
					err = errFind
				}
			}
		default:
			res = fmt.Sprintf("未知操作: %s", op.Action)
		}

		if err != nil {
			results = append(results, fmt.Sprintf("%s失败: %v", getActionNameCN(op.Action), err))
		} else {
			results = append(results, res)
		}
	}

	return results, nil
}

func (s *AdminUsecase) resolveClassID(ctx context.Context, schoolID, deptName, className string) (string, error) {
	if className == "" {
		return "", nil
	}
	depts, err := s.departments.List(ctx, schoolID)
	if err != nil {
		return "", err
	}

	var targetDeptID string
	if deptName != "" {
		for _, d := range depts {
			if d.Name == deptName {
				targetDeptID = d.ID
				break
			}
		}
		if targetDeptID == "" {
			return "", fmt.Errorf("department not found: %s", deptName)
		}
	}

	candidateDepts := depts
	if targetDeptID != "" {
		candidateDepts = []organization.Department{}
		for _, d := range depts {
			if d.ID == targetDeptID {
				candidateDepts = append(candidateDepts, d)
				break
			}
		}
	}

	for _, d := range candidateDepts {
		classes, err := s.classes.ListByDepartment(ctx, schoolID, d.ID)
		if err != nil {
			continue
		}
		for _, c := range classes {
			if c.Name == className {
				return c.ID, nil
			}
		}
	}

	return "", fmt.Errorf("class not found: %s", className)
}
