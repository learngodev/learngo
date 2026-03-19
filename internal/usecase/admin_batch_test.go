package service

import (
	"context"
	"errors"
	"gorm.io/gorm"
	identitybiz "learn-go/internal/biz/identity"
	schoolbiz "learn-go/internal/biz/school"
	sharedbiz "learn-go/internal/biz/shared"
	"reflect"
	"strings"
	"testing"
)

func TestBatchOperateAccountsRequiresIDs(t *testing.T) {
	svc := newTestAdminService(newMemoryAccountRepo())
	_, err := svc.BatchOperateAccounts(context.Background(), AdminBatchOperationInput{
		SchoolID: "school",
		Action:   AdminBatchActionLock,
	})
	if !errors.Is(err, ErrAdminBatchAccountIDsRequired) {
		t.Fatalf("expected ErrAdminBatchAccountIDsRequired, got %v", err)
	}
}

func TestBatchOperateAccountsExecutesActions(t *testing.T) {
	repo := newMemoryAccountRepo()
	repo.accounts["t1"] = &identitybiz.Account{ID: "t1", SchoolID: "school", Role: sharedbiz.RoleTeacher, Status: sharedbiz.AccountStatusActive}
	repo.accounts["s1"] = &identitybiz.Account{ID: "s1", SchoolID: "school", Role: sharedbiz.RoleStudent, Status: sharedbiz.AccountStatusActive}
	svc := newTestAdminService(repo)

	result, err := svc.BatchOperateAccounts(context.Background(), AdminBatchOperationInput{
		SchoolID:   "school",
		AccountIDs: []string{"t1", "s1", "t1", ""},
		Action:     AdminBatchActionLock,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Succeeded) != 2 {
		t.Fatalf("expected 2 succeeded, got %d", len(result.Succeeded))
	}
	if len(result.Failed) != 0 {
		t.Fatalf("expected no failures, got %v", result.Failed)
	}
	if repo.accounts["t1"].Status != sharedbiz.AccountStatusLocked {
		t.Fatalf("expected t1 to be locked")
	}
	if repo.accounts["s1"].Status != sharedbiz.AccountStatusLocked {
		t.Fatalf("expected s1 to be locked")
	}
}

func TestBatchOperateAccountsCollectsFailures(t *testing.T) {
	repo := newMemoryAccountRepo()
	repo.accounts["t1"] = &identitybiz.Account{ID: "t1", SchoolID: "school", Role: sharedbiz.RoleTeacher, Status: sharedbiz.AccountStatusActive}
	repo.accounts["s1"] = &identitybiz.Account{ID: "s1", SchoolID: "other", Role: sharedbiz.RoleStudent, Status: sharedbiz.AccountStatusActive}
	svc := newTestAdminService(repo)

	result, err := svc.BatchOperateAccounts(context.Background(), AdminBatchOperationInput{
		SchoolID:   "school",
		AccountIDs: []string{"t1", "missing", "s1"},
		Action:     AdminBatchActionResetPassword,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result.Succeeded, []string{"t1"}) {
		t.Fatalf("unexpected succeeded set: %v", result.Succeeded)
	}
	if len(result.Failed) != 2 {
		t.Fatalf("expected 2 failures, got %d", len(result.Failed))
	}
	if repo.accounts["t1"].Status != sharedbiz.AccountStatusPasswordResetRequired {
		t.Fatalf("expected password reset status")
	}
	if _, ok := repo.accounts["missing"]; ok {
		t.Fatalf("missing account should not be created")
	}
}

type memoryAccountRepo struct {
	accounts map[string]*identitybiz.Account
}

func newMemoryAccountRepo() *memoryAccountRepo {
	return &memoryAccountRepo{accounts: make(map[string]*identitybiz.Account)}
}

func (m *memoryAccountRepo) Create(ctx context.Context, account *identitybiz.Account) error {
	cp := *account
	m.accounts[account.ID] = &cp
	return nil
}

func (m *memoryAccountRepo) FindByIdentifier(context.Context, string, string) (*identitybiz.Account, error) {
	return nil, gorm.ErrRecordNotFound
}

func (m *memoryAccountRepo) FindByID(ctx context.Context, id string) (*identitybiz.Account, error) {
	if acc, ok := m.accounts[id]; ok {
		cp := *acc
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *memoryAccountRepo) ListByIDs(context.Context, []string) ([]identitybiz.Account, error) {
	return nil, errors.New("not implemented")
}

func (m *memoryAccountRepo) ListByRole(ctx context.Context, schoolID string, role sharedbiz.Role, status sharedbiz.AccountStatus, deptID, classID, courseID string, onlyClassless, onlyDeptless bool, page, size int, query string) ([]identitybiz.Account, int64, error) {
	var result []identitybiz.Account
	for _, acc := range m.accounts {
		if acc.SchoolID == schoolID {
			if query != "" {
				// Mock search by name/identifier
				// Assuming DisplayName is populated in test data
				// But memoryAccountRepo only sets ID, SchoolID, Role, Status in tests usually.
				// We need to ensure test data has DisplayName if we want to test name lookup.
				// For now, let's just check if query matches ID or DisplayName (if present)
				match := false
				if strings.Contains(acc.Identifier, query) {
					match = true
				}
				// We don't have DisplayName in Account struct in this file context,
				// but identitybiz.Account has it.
				if strings.Contains(acc.DisplayName, query) {
					match = true
				}
				if !match {
					continue
				}
			}
			result = append(result, *acc)
		}
	}
	return result, int64(len(result)), nil
}

func (m *memoryAccountRepo) UpdateStatus(ctx context.Context, accountID, schoolID string, status sharedbiz.AccountStatus) error {
	acc, ok := m.accounts[accountID]
	if !ok || acc.SchoolID != schoolID {
		return gorm.ErrRecordNotFound
	}
	acc.Status = status
	return nil
}

func (m *memoryAccountRepo) UpdatePasswordHash(context.Context, string, string) error {
	return nil
}

func (m *memoryAccountRepo) Update(ctx context.Context, account *identitybiz.Account) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *memoryAccountRepo) Delete(ctx context.Context, accountID, schoolID string) error {
	acc, ok := m.accounts[accountID]
	if !ok || acc.SchoolID != schoolID {
		return gorm.ErrRecordNotFound
	}
	delete(m.accounts, accountID)
	return nil
}

var _ identitybiz.AccountRepository = (*memoryAccountRepo)(nil)

type noopTeacherRepo struct{}

func (noopTeacherRepo) Create(context.Context, *identitybiz.Teacher) error { return nil }
func (noopTeacherRepo) GetByNumber(context.Context, string, string) (*identitybiz.Teacher, error) {
	return nil, nil
}
func (noopTeacherRepo) GetByID(context.Context, string) (*identitybiz.Teacher, error) {
	return nil, nil
}
func (noopTeacherRepo) GetByAccountID(context.Context, string) (*identitybiz.Teacher, error) {
	return nil, nil
}
func (noopTeacherRepo) Update(context.Context, *identitybiz.Teacher) error { return nil }

var _ identitybiz.TeacherRepository = (*noopTeacherRepo)(nil)

type noopStudentRepo struct{}

func (noopStudentRepo) Create(context.Context, *identitybiz.Student) error { return nil }
func (noopStudentRepo) GetByNumber(context.Context, string, string) (*identitybiz.Student, error) {
	return nil, nil
}
func (noopStudentRepo) GetByID(context.Context, string) (*identitybiz.Student, error) {
	return nil, nil
}
func (noopStudentRepo) GetByAccountID(context.Context, string) (*identitybiz.Student, error) {
	return nil, nil
}
func (noopStudentRepo) ListByIDs(context.Context, []string) ([]identitybiz.Student, error) {
	return []identitybiz.Student{}, nil
}
func (noopStudentRepo) CountByClassIDs(context.Context, []string) (map[string]int64, error) {
	return map[string]int64{}, nil
}
func (noopStudentRepo) UpdateClassID(context.Context, string, string) error { return nil }
func (noopStudentRepo) ListByClassID(context.Context, string) ([]identitybiz.Student, error) {
	return nil, nil
}
func (noopStudentRepo) ListByDepartmentID(context.Context, string) ([]identitybiz.Student, error) {
	return nil, nil
}
func (noopStudentRepo) Update(context.Context, *identitybiz.Student) error { return nil }

var _ identitybiz.StudentRepository = (*noopStudentRepo)(nil)

type noopDepartmentRepo struct{}

func (noopDepartmentRepo) Create(context.Context, *schoolbiz.Department) error { return nil }
func (noopDepartmentRepo) List(context.Context, string) ([]schoolbiz.Department, error) {
	return nil, nil
}
func (noopDepartmentRepo) GetByID(context.Context, string) (*schoolbiz.Department, error) {
	return &schoolbiz.Department{SchoolID: "school"}, nil
}
func (noopDepartmentRepo) UpdateName(context.Context, string, string, string) error { return nil }
func (noopDepartmentRepo) Delete(context.Context, string, string) error             { return nil }

var _ schoolbiz.DepartmentRepository = (*noopDepartmentRepo)(nil)

type noopClassRepo struct{}

func (noopClassRepo) Create(context.Context, *schoolbiz.Class) error { return nil }
func (noopClassRepo) ListByDepartment(context.Context, string, string) ([]schoolbiz.Class, error) {
	return nil, nil
}
func (noopClassRepo) GetByID(context.Context, string) (*schoolbiz.Class, error) {
	return &schoolbiz.Class{SchoolID: "school"}, nil
}
func (noopClassRepo) ListByIDs(context.Context, []string) ([]schoolbiz.Class, error) { return nil, nil }
func (noopClassRepo) UpdateName(context.Context, string, string, string) error       { return nil }
func (noopClassRepo) Delete(context.Context, string, string) error                   { return nil }

var _ schoolbiz.ClassRepository = (*noopClassRepo)(nil)

type noopTeacherStudentRepo struct{}

func (noopTeacherStudentRepo) BindTeachers(context.Context, string, []string) error { return nil }

var _ identitybiz.TeacherStudentRepository = (*noopTeacherStudentRepo)(nil)

func newTestAdminService(acc identitybiz.AccountRepository) *AdminService {
	return &AdminService{
		accounts:     acc,
		teachers:     noopTeacherRepo{},
		students:     noopStudentRepo{},
		departments:  noopDepartmentRepo{},
		classes:      noopClassRepo{},
		teacherLinks: noopTeacherStudentRepo{},
	}
}
