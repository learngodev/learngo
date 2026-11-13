package gormrepo

import (
	"context"
	"errors"

	"learn-go/internal/domain/roster"

	"gorm.io/gorm"
)

// TeacherStore 基于 GORM 实现 roster.TeacherRepository。
type TeacherStore struct {
	db *gorm.DB
}

// NewTeacherStore 返回新的教师存储。
func NewTeacherStore(db *gorm.DB) *TeacherStore {
	return &TeacherStore{db: db}
}

func (s *TeacherStore) Create(ctx context.Context, teacher *roster.Teacher) error {
	return s.db.WithContext(ctx).Create(teacher).Error
}

func (s *TeacherStore) Update(ctx context.Context, teacher *roster.Teacher) error {
	return s.db.WithContext(ctx).Save(teacher).Error
}

func (s *TeacherStore) GetByNumber(ctx context.Context, schoolID, number string) (*roster.Teacher, error) {
	var teacher roster.Teacher
	if err := s.db.WithContext(ctx).Where("school_id = ? AND number = ?", schoolID, number).First(&teacher).Error; err != nil {
		return nil, notFound(err)
	}
	return &teacher, nil
}

func (s *TeacherStore) GetByID(ctx context.Context, id string) (*roster.Teacher, error) {
	var teacher roster.Teacher
	if err := s.db.WithContext(ctx).First(&teacher, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	return &teacher, nil
}

func (s *TeacherStore) GetByAccountID(ctx context.Context, accountID string) (*roster.Teacher, error) {
	var teacher roster.Teacher
	if err := s.db.WithContext(ctx).Where("account_id = ?", accountID).First(&teacher).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &teacher, nil
}

var _ roster.TeacherRepository = (*TeacherStore)(nil)
