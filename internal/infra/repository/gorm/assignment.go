package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain/coursework"
	"learn-go/internal/domain/storage"

	"gorm.io/gorm"
)

// AssignmentStore 基于 GORM 实现 coursework.AssignmentRepository。
type AssignmentStore struct {
	db *gorm.DB
}

func NewAssignmentStore(db *gorm.DB) *AssignmentStore {
	return &AssignmentStore{db: db}
}

func (s *AssignmentStore) Create(ctx context.Context, assignment *coursework.Assignment, questions []coursework.AssignmentQuestion, attachments []coursework.AssignmentAttachment) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(assignment).Error; err != nil {
			return err
		}
		for i := range questions {
			questions[i].AssignmentID = assignment.ID
		}
		if len(questions) > 0 {
			if err := tx.Create(&questions).Error; err != nil {
				return err
			}
		}
		for i := range attachments {
			attachments[i].AssignmentID = assignment.ID
		}
		if len(attachments) > 0 {
			if err := tx.Create(&attachments).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *AssignmentStore) Get(ctx context.Context, id string) (*coursework.Assignment, []coursework.AssignmentQuestion, []storage.File, error) {
	var assignment coursework.Assignment
	if err := s.db.WithContext(ctx).First(&assignment, "id = ?", id).Error; err != nil {
		return nil, nil, nil, notFound(err)
	}
	var questions []coursework.AssignmentQuestion
	if err := s.db.WithContext(ctx).Where("assignment_id = ?", assignment.ID).Order("order_index").Find(&questions).Error; err != nil {
		return nil, nil, nil, err
	}
	var files []storage.File
	if err := s.db.WithContext(ctx).
		Table("files").
		Joins("JOIN assignment_attachments ON assignment_attachments.file_id = files.id").
		Where("assignment_attachments.assignment_id = ?", assignment.ID).
		Find(&files).Error; err != nil {
		return nil, nil, nil, err
	}
	return &assignment, questions, files, nil
}

func (s *AssignmentStore) ListByClass(ctx context.Context, classID string, limit int, types []coursework.AssignmentType) ([]coursework.Assignment, error) {
	query := s.db.WithContext(ctx).Where("class_id = ?", classID)
	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var assignments []coursework.Assignment
	if err := query.
		Order("COALESCE(due_at, start_at, created_at) ASC").
		Limit(limit).
		Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

func (s *AssignmentStore) ListDueBetween(ctx context.Context, classID string, start, end time.Time, types []coursework.AssignmentType) ([]coursework.Assignment, error) {
	query := s.db.WithContext(ctx).Where("class_id = ?", classID)
	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	if !start.IsZero() {
		query = query.Where("due_at IS NOT NULL AND due_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("due_at < ?", end)
	}

	var assignments []coursework.Assignment
	if err := query.Order("due_at ASC NULLS LAST").Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

func (s *AssignmentStore) ListByTeacher(ctx context.Context, teacherID string, limit int, classID string, types []coursework.AssignmentType) ([]coursework.Assignment, error) {
	query := s.db.WithContext(ctx).Where("teacher_id = ?", teacherID)
	if classID != "" {
		query = query.Where("class_id = ?", classID)
	}
	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var assignments []coursework.Assignment
	if err := query.Order("COALESCE(due_at, start_at, created_at) DESC").Limit(limit).Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

func (s *AssignmentStore) ListDueBetweenByTeacher(ctx context.Context, teacherID string, start, end time.Time, types []coursework.AssignmentType) ([]coursework.Assignment, error) {
	query := s.db.WithContext(ctx).Where("teacher_id = ?", teacherID)
	if len(types) > 0 {
		query = query.Where("type IN ?", types)
	}
	if !start.IsZero() {
		query = query.Where("due_at IS NOT NULL AND due_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("due_at < ?", end)
	}

	var assignments []coursework.Assignment
	if err := query.Order("due_at ASC NULLS LAST").Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

func (s *AssignmentStore) Search(ctx context.Context, teacherID string, q string) ([]coursework.Assignment, error) {
	var assignments []coursework.Assignment
	if err := s.db.WithContext(ctx).
		Where("teacher_id = ? AND (title LIKE ? OR description LIKE ?)", teacherID, "%"+q+"%", "%"+q+"%").
		Order("created_at DESC").
		Limit(20).
		Find(&assignments).Error; err != nil {
		return nil, err
	}
	return assignments, nil
}

func (s *AssignmentStore) Update(ctx context.Context, assignment *coursework.Assignment) error {
	return s.db.WithContext(ctx).Save(assignment).Error
}

var _ coursework.AssignmentRepository = (*AssignmentStore)(nil)
