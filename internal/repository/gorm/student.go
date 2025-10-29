package gormrepo

import (
	"context"
	"errors"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// StudentStore implements repository.StudentRepository using GORM.
type StudentStore struct {
	db *gorm.DB
}

// NewStudentStore returns a new student store.
func NewStudentStore(db *gorm.DB) *StudentStore {
	return &StudentStore{db: db}
}

func (s *StudentStore) Create(ctx context.Context, student *domain.Student) error {
	return s.db.WithContext(ctx).Create(student).Error
}

func (s *StudentStore) GetByNumber(ctx context.Context, schoolID, number string) (*domain.Student, error) {
	var student domain.Student
	if err := s.db.WithContext(ctx).Where("school_id = ? AND number = ?", schoolID, number).First(&student).Error; err != nil {
		return nil, err
	}
	return &student, nil
}

func (s *StudentStore) GetByID(ctx context.Context, id string) (*domain.Student, error) {
	var student domain.Student
	if err := s.db.WithContext(ctx).First(&student, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &student, nil
}

func (s *StudentStore) GetByAccountID(ctx context.Context, accountID string) (*domain.Student, error) {
	var student domain.Student
	if err := s.db.WithContext(ctx).Where("account_id = ?", accountID).First(&student).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &student, nil
}

var _ repository.StudentRepository = (*StudentStore)(nil)
