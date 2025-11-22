package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// StudentReminderStore implements repository.StudentReminderRepository using GORM.
type StudentReminderStore struct {
	db *gorm.DB
}

// NewStudentReminderStore constructs a reminder store.
func NewStudentReminderStore(db *gorm.DB) *StudentReminderStore {
	return &StudentReminderStore{db: db}
}

func (s *StudentReminderStore) ListByStudent(ctx context.Context, studentID string) ([]domain.StudentReminder, error) {
	reminders := make([]domain.StudentReminder, 0)
	if err := s.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Order("created_at DESC").
		Find(&reminders).Error; err != nil {
		return nil, err
	}
	return reminders, nil
}

func (s *StudentReminderStore) GetByID(ctx context.Context, id string) (*domain.StudentReminder, error) {
	var reminder domain.StudentReminder
	if err := s.db.WithContext(ctx).First(&reminder, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &reminder, nil
}

func (s *StudentReminderStore) Create(ctx context.Context, reminder *domain.StudentReminder) error {
	return s.db.WithContext(ctx).Create(reminder).Error
}

func (s *StudentReminderStore) UpdateFields(ctx context.Context, id string, studentID string, updates map[string]any) (*domain.StudentReminder, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	updates["updated_at"] = time.Now()

	result := s.db.WithContext(ctx).
		Model(&domain.StudentReminder{}).
		Where("id = ? AND student_id = ?", id, studentID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return s.GetByID(ctx, id)
}

func (s *StudentReminderStore) Delete(ctx context.Context, id string, studentID string) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND student_id = ?", id, studentID).
		Delete(&domain.StudentReminder{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *StudentReminderStore) SetCompletion(ctx context.Context, id string, studentID string, completed bool, timestamp *time.Time) (*domain.StudentReminder, error) {
	updates := map[string]any{
		"completed_at": reminderCompletionValue(completed, timestamp),
		"updated_at":   time.Now(),
	}

	result := s.db.WithContext(ctx).
		Model(&domain.StudentReminder{}).
		Where("id = ? AND student_id = ?", id, studentID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return s.GetByID(ctx, id)
}

func (s *StudentReminderStore) MarkBatchCompleted(ctx context.Context, studentID string, ids []string, completed bool, timestamp *time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	updates := map[string]any{
		"completed_at": reminderCompletionValue(completed, timestamp),
		"updated_at":   time.Now(),
	}
	result := s.db.WithContext(ctx).
		Model(&domain.StudentReminder{}).
		Where("student_id = ? AND id IN ?", studentID, ids).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *StudentReminderStore) MarkAllCompleted(ctx context.Context, studentID string, completed bool, timestamp *time.Time) error {
	return s.db.WithContext(ctx).
		Model(&domain.StudentReminder{}).
		Where("student_id = ?", studentID).
		Updates(map[string]any{
			"completed_at": reminderCompletionValue(completed, timestamp),
			"updated_at":   time.Now(),
		}).Error
}

func reminderCompletionValue(completed bool, timestamp *time.Time) any {
	if !completed {
		return nil
	}
	if timestamp != nil {
		return *timestamp
	}
	return time.Now()
}

var _ repository.StudentReminderRepository = (*StudentReminderStore)(nil)
