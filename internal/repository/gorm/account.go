package gormrepo

import (
	"context"
	"strings"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// AccountStore implements repository.AccountRepository using GORM.
type AccountStore struct {
	db *gorm.DB
}

// NewAccountStore creates a new AccountStore.
func NewAccountStore(db *gorm.DB) *AccountStore {
	return &AccountStore{db: db}
}

func (s *AccountStore) Create(ctx context.Context, account *domain.Account) error {
	return s.db.WithContext(ctx).Create(account).Error
}

func (s *AccountStore) Update(ctx context.Context, account *domain.Account) error {
	return s.db.WithContext(ctx).Save(account).Error
}

func (s *AccountStore) FindByIdentifier(ctx context.Context, schoolID, identifier string) (*domain.Account, error) {
	var account domain.Account
	if err := s.db.WithContext(ctx).Where("school_id = ? AND identifier = ?", schoolID, identifier).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *AccountStore) FindByID(ctx context.Context, id string) (*domain.Account, error) {
	var account domain.Account
	if err := s.db.WithContext(ctx).First(&account, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *AccountStore) ListByIDs(ctx context.Context, ids []string) ([]domain.Account, error) {
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

func (s *AccountStore) ListByRole(
	ctx context.Context,
	schoolID string,
	role domain.Role,
	status domain.AccountStatus,
	departmentID string,
	classID string,
	courseID string,
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
	joinedTeachers := false

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
	joinTeachers := func() {
		if !joinedTeachers {
			base = base.Joins("LEFT JOIN teachers ON teachers.account_id = accounts.id")
			joinedTeachers = true
		}
	}

	if classID != "" {
		if role == domain.RoleTeacher {
			joinTeachers()
			base = base.Where(
				"teachers.id IN (?) OR teachers.id IN (?)",
				s.db.Table("course_schedules").Select("teacher_id").Where("class_id = ?", classID),
				s.db.Table("class_teachers").Select("teacher_id").Where("class_id = ?", classID),
			)
		} else {
			joinStudents()
			base = base.Where("students.class_id = ?", classID)
		}
	}
	if departmentID != "" {
		if role == domain.RoleTeacher {
			joinTeachers()
			base = base.Where(
				"teachers.id IN (?) OR teachers.id IN (?)",
				s.db.Table("course_schedules").
					Select("course_schedules.teacher_id").
					Joins("JOIN classes ON classes.id = course_schedules.class_id").
					Where("classes.department_id = ? AND course_schedules.teacher_id IS NOT NULL", departmentID),
				s.db.Table("class_teachers").
					Select("class_teachers.teacher_id").
					Joins("JOIN classes ON classes.id = class_teachers.class_id").
					Where("classes.department_id = ?", departmentID),
			)
		} else {
			joinClasses()
			base = base.Where("classes.department_id = ?", departmentID)
		}
	}
	if courseID != "" {
		if role == domain.RoleTeacher {
			joinTeachers()
			base = base.Where(
				"teachers.id IN (?)",
				s.db.Table("course_schedules").Select("teacher_id").Where("course_id = ? AND teacher_id IS NOT NULL", courseID),
			)
		} else {
			joinStudents()
			base = base.Joins("JOIN course_schedules ON course_schedules.class_id = students.class_id").
				Where("course_schedules.course_id = ?", courseID)
		}
	}

	if onlyClassless {
		if role == domain.RoleTeacher {
			joinTeachers()
			base = base.Where(
				"teachers.id NOT IN (?) AND teachers.id NOT IN (?)",
				s.db.Table("course_schedules").
					Select("teacher_id").
					Where("teacher_id IS NOT NULL AND class_id <> ''"),
				s.db.Table("class_teachers").
					Select("teacher_id"),
			)
		} else {
			joinStudents()
			base = base.Where("(students.class_id IS NULL OR students.class_id = '')")
		}
	}

	if onlyDepartmentless {
		if role == domain.RoleTeacher {
			joinTeachers()
			base = base.Where(
				"teachers.id NOT IN (?) AND teachers.id NOT IN (?)",
				s.db.Table("course_schedules").
					Select("course_schedules.teacher_id").
					Joins("JOIN classes ON classes.id = course_schedules.class_id").
					Where("course_schedules.teacher_id IS NOT NULL AND classes.department_id IS NOT NULL AND classes.department_id <> ''"),
				s.db.Table("class_teachers").
					Select("class_teachers.teacher_id").
					Joins("JOIN classes ON classes.id = class_teachers.class_id").
					Where("classes.department_id IS NOT NULL AND classes.department_id <> ''"),
			)
		} else {
			joinClasses()
			base = base.Where("(classes.department_id IS NULL OR classes.department_id = '')")
		}
	}

	countQuery := base.Session(&gorm.Session{}).Distinct("accounts.id")
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	dataQuery := base.Session(&gorm.Session{}).
		Select("accounts.*").
		Distinct().
		Order("accounts.created_at DESC").
		Offset(offset).
		Limit(size)

	if err := dataQuery.Find(&accounts).Error; err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}

func (s *AccountStore) UpdateStatus(ctx context.Context, accountID, schoolID string, status domain.AccountStatus) error {
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

func (s *AccountStore) UpdatePasswordHash(ctx context.Context, accountID string, passwordHash string) error {
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

func (s *AccountStore) Delete(ctx context.Context, accountID, schoolID string) error {
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

var _ repository.AccountRepository = (*AccountStore)(nil)
