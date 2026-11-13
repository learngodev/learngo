package usecase

import (
	"context"
	"errors"
	"learn-go/internal/domain"
	"reflect"
	"strings"
	"testing"

	"learn-go/internal/domain/identity"
	"learn-go/internal/domain/organization"
	"learn-go/internal/domain/roster"
)

func TestBatchOperateAccountsRequiresIDs(t *testing.T) {
	svc := newTestAdminUsecase(newMemoryAccountRepo())
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
	repo.accounts["t1"] = &identity.Account{ID: "t1", SchoolID: "school", Role: identity.RoleTeacher, Status: identity.AccountStatusActive}
	repo.accounts["s1"] = &identity.Account{ID: "s1", SchoolID: "school", Role: identity.RoleStudent, Status: identity.AccountStatusActive}
	svc := newTestAdminUsecase(repo)

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
	if repo.accounts["t1"].Status != identity.AccountStatusLocked {
		t.Fatalf("expected t1 to be locked")
	}
	if repo.accounts["s1"].Status != identity.AccountStatusLocked {
		t.Fatalf("expected s1 to be locked")
	}
}

func TestBatchOperateAccountsCollectsFailures(t *testing.T) {
	repo := newMemoryAccountRepo()
	repo.accounts["t1"] = &identity.Account{ID: "t1", SchoolID: "school", Role: identity.RoleTeacher, Status: identity.AccountStatusActive}
	repo.accounts["s1"] = &identity.Account{ID: "s1", SchoolID: "other", Role: identity.RoleStudent, Status: identity.AccountStatusActive}
	svc := newTestAdminUsecase(repo)

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
	if repo.accounts["t1"].Status != identity.AccountStatusPasswordResetRequired {
		t.Fatalf("expected password reset status")
	}
	if _, ok := repo.accounts["missing"]; ok {
		t.Fatalf("missing account should not be created")
	}
}

type memoryAccountRepo struct {
	accounts map[string]*identity.Account
}

func newMemoryAccountRepo() *memoryAccountRepo {
	return &memoryAccountRepo{accounts: make(map[string]*identity.Account)}
}

func (m *memoryAccountRepo) Create(ctx context.Context, account *identity.Account) error {
	cp := *account
	m.accounts[account.ID] = &cp
	return nil
}

func (m *memoryAccountRepo) FindByIdentifier(context.Context, string, string) (*identity.Account, error) {
	return nil, domain.ErrNotFound
}

func (m *memoryAccountRepo) FindByID(ctx context.Context, id string) (*identity.Account, error) {
	if acc, ok := m.accounts[id]; ok {
		cp := *acc
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (m *memoryAccountRepo) ListByIDs(context.Context, []string) ([]identity.Account, error) {
	return nil, errors.New("not implemented")
}

func (m *memoryAccountRepo) ListByRole(ctx context.Context, schoolID string, role identity.Role, status identity.AccountStatus, deptID, classID, courseID string, onlyClassless, onlyDeptless bool, page, size int, query string) ([]identity.Account, int64, error) {
	var result []identity.Account
	for _, acc := range m.accounts {
		if acc.SchoolID == schoolID {
			if query != "" {
				// 模拟按姓名/编号搜索
				// 假定测试数据中已填充 DisplayName
				// 但测试中 memoryAccountRepo 通常只设置 ID、SchoolID、Role、Status。
				// 若要测试按姓名查找，需确保测试数据带有 DisplayName。
				// 目前仅检查查询是否匹配 ID 或 DisplayName（若存在）
				match := false
				if strings.Contains(acc.Identifier, query) {
					match = true
				}
				// 本文件上下文中的 Account 结构体没有 DisplayName，
				// 但 identity.Account 有该字段。
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

func (m *memoryAccountRepo) UpdateStatus(ctx context.Context, accountID, schoolID string, status identity.AccountStatus) error {
	acc, ok := m.accounts[accountID]
	if !ok || acc.SchoolID != schoolID {
		return domain.ErrNotFound
	}
	acc.Status = status
	return nil
}

func (m *memoryAccountRepo) UpdatePasswordHash(context.Context, string, string) error {
	return nil
}

func (m *memoryAccountRepo) Update(ctx context.Context, account *identity.Account) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *memoryAccountRepo) Delete(ctx context.Context, accountID, schoolID string) error {
	acc, ok := m.accounts[accountID]
	if !ok || acc.SchoolID != schoolID {
		return domain.ErrNotFound
	}
	delete(m.accounts, accountID)
	return nil
}

var _ identity.AccountRepository = (*memoryAccountRepo)(nil)

type noopTeacherRepo struct{}

func (noopTeacherRepo) Create(context.Context, *roster.Teacher) error { return nil }
func (noopTeacherRepo) GetByNumber(context.Context, string, string) (*roster.Teacher, error) {
	return nil, nil
}
func (noopTeacherRepo) GetByID(context.Context, string) (*roster.Teacher, error) { return nil, nil }
func (noopTeacherRepo) GetByAccountID(context.Context, string) (*roster.Teacher, error) {
	return nil, nil
}
func (noopTeacherRepo) Update(context.Context, *roster.Teacher) error { return nil }

var _ roster.TeacherRepository = (*noopTeacherRepo)(nil)

type noopStudentRepo struct{}

func (noopStudentRepo) Create(context.Context, *roster.Student) error { return nil }
func (noopStudentRepo) GetByNumber(context.Context, string, string) (*roster.Student, error) {
	return nil, nil
}
func (noopStudentRepo) GetByID(context.Context, string) (*roster.Student, error) { return nil, nil }
func (noopStudentRepo) GetByAccountID(context.Context, string) (*roster.Student, error) {
	return nil, nil
}
func (noopStudentRepo) ListByIDs(context.Context, []string) ([]roster.Student, error) {
	return []roster.Student{}, nil
}
func (noopStudentRepo) CountByClassIDs(context.Context, []string) (map[string]int64, error) {
	return map[string]int64{}, nil
}
func (noopStudentRepo) UpdateClassID(context.Context, string, string) error { return nil }
func (noopStudentRepo) ListByClassID(context.Context, string) ([]roster.Student, error) {
	return nil, nil
}
func (noopStudentRepo) ListByDepartmentID(context.Context, string) ([]roster.Student, error) {
	return nil, nil
}
func (noopStudentRepo) Update(context.Context, *roster.Student) error { return nil }

var _ roster.StudentRepository = (*noopStudentRepo)(nil)

type noopDepartmentRepo struct{}

func (noopDepartmentRepo) Create(context.Context, *organization.Department) error { return nil }
func (noopDepartmentRepo) List(context.Context, string) ([]organization.Department, error) {
	return nil, nil
}
func (noopDepartmentRepo) GetByID(context.Context, string) (*organization.Department, error) {
	return &organization.Department{SchoolID: "school"}, nil
}
func (noopDepartmentRepo) UpdateName(context.Context, string, string, string) error { return nil }
func (noopDepartmentRepo) Delete(context.Context, string, string) error             { return nil }

var _ organization.DepartmentRepository = (*noopDepartmentRepo)(nil)

type noopClassRepo struct{}

func (noopClassRepo) Create(context.Context, *organization.Class) error { return nil }
func (noopClassRepo) ListByDepartment(context.Context, string, string) ([]organization.Class, error) {
	return nil, nil
}
func (noopClassRepo) GetByID(context.Context, string) (*organization.Class, error) {
	return &organization.Class{SchoolID: "school"}, nil
}
func (noopClassRepo) ListByIDs(context.Context, []string) ([]organization.Class, error) {
	return nil, nil
}
func (noopClassRepo) UpdateName(context.Context, string, string, string) error { return nil }
func (noopClassRepo) Delete(context.Context, string, string) error             { return nil }

var _ organization.ClassRepository = (*noopClassRepo)(nil)

type noopTeacherStudentRepo struct{}

func (noopTeacherStudentRepo) BindTeachers(context.Context, string, []string) error { return nil }

var _ roster.TeacherStudentRepository = (*noopTeacherStudentRepo)(nil)

func newTestAdminUsecase(acc identity.AccountRepository) *AdminUsecase {
	return &AdminUsecase{
		accounts:     acc,
		teachers:     noopTeacherRepo{},
		students:     noopStudentRepo{},
		departments:  noopDepartmentRepo{},
		classes:      noopClassRepo{},
		teacherLinks: noopTeacherStudentRepo{},
		hasher:       plainHasher{},
	}
}

type plainHasher struct{}

func (plainHasher) Hash(plain string) (string, error) { return "hashed:" + plain, nil }
func (plainHasher) Compare(hash, plain string) error {
	if hash != "hashed:"+plain {
		return errors.New("password mismatch")
	}
	return nil
}

var _ identity.PasswordHasher = plainHasher{}
