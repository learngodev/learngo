package gormrepo

import (
	"context"
	"errors"
	"gorm.io/gorm"
	identitybiz "learn-go/internal/biz/identity"
)

// TeacherStore implements identitybiz.TeacherRepository using GORM.
type TeacherStore struct {
	db *gorm.DB
}

// NewTeacherStore returns a new teacher store.
func NewTeacherStore(db *gorm.DB) *TeacherStore {
	return &TeacherStore{db: db}
}

func (s *TeacherStore) Create(ctx context.Context, teacher *identitybiz.Teacher) error {
	return s.db.WithContext(ctx).Create(teacher).Error
}

func (s *TeacherStore) Update(ctx context.Context, teacher *identitybiz.Teacher) error {
	return s.db.WithContext(ctx).Save(teacher).Error
}

func (s *TeacherStore) GetByNumber(ctx context.Context, schoolID, number string) (*identitybiz.Teacher, error) {
	var teacher identitybiz.Teacher
	if err := s.db.WithContext(ctx).Where("school_id = ? AND number = ?", schoolID, number).First(&teacher).Error; err != nil {
		return nil, err
	}
	return &teacher, nil
}

func (s *TeacherStore) GetByID(ctx context.Context, id string) (*identitybiz.Teacher, error) {
	var teacher identitybiz.Teacher
	if err := s.db.WithContext(ctx).First(&teacher, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &teacher, nil
}

func (s *TeacherStore) GetByAccountID(ctx context.Context, accountID string) (*identitybiz.Teacher, error) {
	var teacher identitybiz.Teacher
	if err := s.db.WithContext(ctx).Where("account_id = ?", accountID).First(&teacher).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &teacher, nil
}

var _ identitybiz.TeacherRepository = (*TeacherStore)(nil)
