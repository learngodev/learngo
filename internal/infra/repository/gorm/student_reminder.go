package gormrepo

import (
	"context"
	"learn-go/internal/domain"
	"time"

	"learn-go/internal/domain/roster"

	"gorm.io/gorm"
)

// StudentReminderStore 基于 GORM 实现 roster.StudentReminderRepository。
type StudentReminderStore struct {
	db *gorm.DB
}

// NewStudentReminderStore 构造提醒事项存储。
func NewStudentReminderStore(db *gorm.DB) *StudentReminderStore {
	return &StudentReminderStore{db: db}
}

func (s *StudentReminderStore) ListByStudent(ctx context.Context, studentID string) ([]roster.StudentReminder, error) {
	reminders := make([]roster.StudentReminder, 0)
	if err := s.db.WithContext(ctx).
		Where("student_id = ?", studentID).
		Order("created_at DESC").
		Find(&reminders).Error; err != nil {
		return nil, err
	}
	return reminders, nil
}

func (s *StudentReminderStore) GetByID(ctx context.Context, id string) (*roster.StudentReminder, error) {
	var reminder roster.StudentReminder
	if err := s.db.WithContext(ctx).First(&reminder, "id = ?", id).Error; err != nil {
		return nil, notFound(err)
	}
	return &reminder, nil
}

func (s *StudentReminderStore) Create(ctx context.Context, reminder *roster.StudentReminder) error {
	return s.db.WithContext(ctx).Create(reminder).Error
}

func (s *StudentReminderStore) UpdateFields(ctx context.Context, id string, studentID string, updates map[string]any) (*roster.StudentReminder, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	updates["updated_at"] = time.Now()

	result := s.db.WithContext(ctx).
		Model(&roster.StudentReminder{}).
		Where("id = ? AND student_id = ?", id, studentID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	return s.GetByID(ctx, id)
}

func (s *StudentReminderStore) Delete(ctx context.Context, id string, studentID string) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND student_id = ?", id, studentID).
		Delete(&roster.StudentReminder{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *StudentReminderStore) SetCompletion(ctx context.Context, id string, studentID string, completed bool, timestamp *time.Time) (*roster.StudentReminder, error) {
	updates := map[string]any{
		"completed_at": reminderCompletionValue(completed, timestamp),
		"updated_at":   time.Now(),
	}

	result := s.db.WithContext(ctx).
		Model(&roster.StudentReminder{}).
		Where("id = ? AND student_id = ?", id, studentID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrNotFound
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
		Model(&roster.StudentReminder{}).
		Where("student_id = ? AND id IN ?", studentID, ids).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *StudentReminderStore) MarkAllCompleted(ctx context.Context, studentID string, completed bool, timestamp *time.Time) error {
	return s.db.WithContext(ctx).
		Model(&roster.StudentReminder{}).
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

var _ roster.StudentReminderRepository = (*StudentReminderStore)(nil)
