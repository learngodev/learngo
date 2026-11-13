package gormrepo

import (
	"context"
	"errors"

	"learn-go/internal/domain/roster"

	"gorm.io/gorm"
)

// StudentStore 基于 GORM 实现 roster.StudentRepository。
type StudentStore struct {
	db *gorm.DB
}

// NewStudentStore 返回新的学生存储。
func NewStudentStore(db *gorm.DB) *StudentStore {
	return &StudentStore{db: db}
}

func (s *StudentStore) Create(ctx context.Context, student *roster.Student) error {
	return s.db.WithContext(ctx).Create(student).Error
}

func (s *StudentStore) Update(ctx context.Context, student *roster.Student) error {
	return s.db.WithContext(ctx).Save(student).Error
}

func (s *StudentStore) GetByNumber(ctx context.Context, schoolID, number string) (*roster.Student, error) {
	var student roster.Student
	if err := s.db.WithContext(ctx).
		Table("students").
		Select("students.*, accounts.display_name as name").
		Joins("LEFT JOIN accounts ON accounts.id = students.account_id").
		Where("students.school_id = ? AND students.number = ?", schoolID, number).
		First(&student).Error; err != nil {
		return nil, notFound(err)
	}
	return &student, nil
}

func (s *StudentStore) GetByID(ctx context.Context, id string) (*roster.Student, error) {
	var student roster.Student
	if err := s.db.WithContext(ctx).
		Table("students").
		Select("students.*, accounts.display_name as name").
		Joins("LEFT JOIN accounts ON accounts.id = students.account_id").
		Where("students.id = ?", id).
		First(&student).Error; err != nil {
		return nil, notFound(err)
	}
	return &student, nil
}

func (s *StudentStore) GetByAccountID(ctx context.Context, accountID string) (*roster.Student, error) {
	var student roster.Student
	if err := s.db.WithContext(ctx).
		Table("students").
		Select("students.*, accounts.display_name as name").
		Joins("LEFT JOIN accounts ON accounts.id = students.account_id").
		Where("students.account_id = ?", accountID).
		First(&student).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &student, nil
}

func (s *StudentStore) ListByIDs(ctx context.Context, ids []string) ([]roster.Student, error) {
	if len(ids) == 0 {
		return []roster.Student{}, nil
	}
	var students []roster.Student
	if err := s.db.WithContext(ctx).
		Table("students").
		Select("students.*, accounts.display_name as name").
		Joins("LEFT JOIN accounts ON accounts.id = students.account_id").
		Where("students.id IN ?", ids).
		Find(&students).Error; err != nil {
		return nil, err
	}
	return students, nil
}

func (s *StudentStore) CountByClassIDs(ctx context.Context, classIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	if len(classIDs) == 0 {
		return result, nil
	}

	type row struct {
		ClassID string
		Count   int64
	}
	var rows []row
	if err := s.db.WithContext(ctx).
		Model(&roster.Student{}).
		Select("class_id", "COUNT(*) AS count").
		Where("class_id IN ?", classIDs).
		Group("class_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ClassID] = row.Count
	}
	return result, nil
}

func (s *StudentStore) UpdateClassID(ctx context.Context, studentID string, classID string) error {
	if classID == "" {
		return s.db.WithContext(ctx).Model(&roster.Student{}).Where("id = ?", studentID).Update("class_id", nil).Error
	}
	return s.db.WithContext(ctx).Model(&roster.Student{}).Where("id = ?", studentID).Update("class_id", classID).Error
}

func (s *StudentStore) ListByClassID(ctx context.Context, classID string) ([]roster.Student, error) {
	var students []roster.Student
	if err := s.db.WithContext(ctx).
		Table("students").
		Select("students.*, accounts.display_name as name").
		Joins("LEFT JOIN accounts ON accounts.id = students.account_id").
		Where("students.class_id = ?", classID).
		Find(&students).Error; err != nil {
		return nil, err
	}
	return students, nil
}

func (s *StudentStore) ListByDepartmentID(ctx context.Context, departmentID string) ([]roster.Student, error) {
	var students []roster.Student
	// 关联 classes 表以按 department_id 筛选
	if err := s.db.WithContext(ctx).
		Joins("JOIN classes ON classes.id = students.class_id").
		Where("classes.department_id = ?", departmentID).
		Find(&students).Error; err != nil {
		return nil, err
	}
	return students, nil
}

var _ roster.StudentRepository = (*StudentStore)(nil)
