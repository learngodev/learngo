package gormrepo

import (
	"context"
	"strings"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// Store implements repository interfaces with GORM.
type Store struct {
	db *gorm.DB
}

// New creates a Store instance.
func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func (s *Store) Create(ctx context.Context, account *domain.Account) error {
	return s.db.WithContext(ctx).Create(account).Error
}

func (s *Store) FindByIdentifier(ctx context.Context, schoolID, identifier string) (*domain.Account, error) {
	var account domain.Account
	if err := s.db.WithContext(ctx).Where("school_id = ? AND identifier = ?", schoolID, identifier).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *Store) FindByID(ctx context.Context, id string) (*domain.Account, error) {
	var account domain.Account
	if err := s.db.WithContext(ctx).First(&account, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *Store) ListByIDs(ctx context.Context, ids []string) ([]domain.Account, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	uniq := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		uniq = append(uniq, trimmed)
	}

	if len(uniq) == 0 {
		return nil, nil
	}

	var accounts []domain.Account
	if err := s.db.WithContext(ctx).Where("id IN ?", uniq).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (s *Store) ListByRole(
	ctx context.Context,
	schoolID string,
	role domain.Role,
	status domain.AccountStatus,
	departmentID string,
	classID string,
	onlyClassless bool,
	onlyDepartmentless bool,
	page int,
	size int,
	search string,
) ([]domain.Account, int64, error) {
	var (
		accounts []domain.Account
		total    int64
	)

	base := s.db.WithContext(ctx).
		Table("accounts").
		Where("accounts.school_id = ?", schoolID)

	if role != "" {
		base = base.Where("accounts.role = ?", role)
	} else {
		base = base.Where("accounts.role IN ?", []domain.Role{domain.RoleTeacher, domain.RoleStudent})
	}

	if status != "" {
		base = base.Where("accounts.status = ?", status)
	}

	if trimmed := strings.TrimSpace(search); trimmed != "" {
		like := "%" + trimmed + "%"
		base = base.Where(
			"accounts.identifier LIKE ? OR accounts.display_name LIKE ?",
			like,
			like,
		)
	}

	joinedStudents := false
	joinedClasses := false
	joinStudents := func() {
		if !joinedStudents {
			base = base.Joins("LEFT JOIN students ON students.account_id = accounts.id")
			joinedStudents = true
		}
	}
	joinClasses := func() {
		joinStudents()
		if !joinedClasses {
			base = base.Joins("LEFT JOIN classes ON classes.id = students.class_id")
			joinedClasses = true
		}
	}

	if classID != "" {
		joinStudents()
		base = base.Where("students.class_id = ?", classID)
	}
	if departmentID != "" {
		joinClasses()
		base = base.Where("classes.department_id = ?", departmentID)
	}

	if onlyClassless {
		joinStudents()
		base = base.Where("(students.class_id IS NULL OR students.class_id = '')")
	}

	if onlyDepartmentless {
		joinClasses()
		base = base.Where("(classes.department_id IS NULL OR classes.department_id = '')")
	}

	countQuery := base.Session(&gorm.Session{}).Distinct("accounts.id")
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	dataQuery := base.Session(&gorm.Session{}).
		Select("accounts.*").
		Distinct("accounts.id").
		Order("accounts.created_at DESC").
		Offset(offset).
		Limit(size)

	if err := dataQuery.Find(&accounts).Error; err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}

func (s *Store) UpdateStatus(ctx context.Context, accountID, schoolID string, status domain.AccountStatus) error {
	result := s.db.WithContext(ctx).
		Model(&domain.Account{}).
		Where("id = ? AND school_id = ?", accountID, schoolID).
		Updates(map[string]any{"status": status})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) UpdatePasswordHash(ctx context.Context, accountID string, passwordHash string) error {
	result := s.db.WithContext(ctx).
		Model(&domain.Account{}).
		Where("id = ?", accountID).
		Updates(map[string]any{"password_hash": passwordHash})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, accountID, schoolID string) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND school_id = ?", accountID, schoolID).
		Delete(&domain.Account{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Ensure Store satisfies interfaces at compile time.
var _ repository.AccountRepository = (*Store)(nil)
