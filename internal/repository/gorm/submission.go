package gormrepo

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// SubmissionStore implements repository.SubmissionRepository with GORM.
type SubmissionStore struct {
	db *gorm.DB
}

func NewSubmissionStore(db *gorm.DB) *SubmissionStore {
	return &SubmissionStore{db: db}
}

func (s *SubmissionStore) CreateOrUpdate(ctx context.Context, submission *domain.AssignmentSubmission, items []domain.SubmissionItem) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if submission.ID == "" {
			if err := tx.Create(submission).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Save(submission).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("submission_id = ?", submission.ID).Delete(&domain.SubmissionItem{}).Error; err != nil {
			return err
		}

		for i := range items {
			items[i].SubmissionID = submission.ID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SubmissionStore) ListByAssignment(ctx context.Context, assignmentID string) ([]domain.AssignmentSubmission, error) {
	var submissions []domain.AssignmentSubmission
	if err := s.db.WithContext(ctx).Where("assignment_id = ?", assignmentID).Order("updated_at DESC").Find(&submissions).Error; err != nil {
		return nil, err
	}
	return submissions, nil
}

func (s *SubmissionStore) ListItemsBySubmissionIDs(ctx context.Context, submissionIDs []string) ([]domain.SubmissionItem, error) {
	if len(submissionIDs) == 0 {
		return nil, nil
	}
	var items []domain.SubmissionItem
	if err := s.db.WithContext(ctx).Where("submission_id IN ?", submissionIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SubmissionStore) GetByAssignmentAndStudent(ctx context.Context, assignmentID, studentID string) (*domain.AssignmentSubmission, []domain.SubmissionItem, error) {
	var submission domain.AssignmentSubmission
	if err := s.db.WithContext(ctx).
		Where("assignment_id = ? AND student_id = ?", assignmentID, studentID).
		First(&submission).Error; err != nil {
		return nil, nil, err
	}
	var items []domain.SubmissionItem
	if err := s.db.WithContext(ctx).Where("submission_id = ?", submission.ID).Find(&items).Error; err != nil {
		return nil, nil, err
	}
	return &submission, items, nil
}

func (s *SubmissionStore) GetByID(ctx context.Context, submissionID string) (*domain.AssignmentSubmission, []domain.SubmissionItem, error) {
	var submission domain.AssignmentSubmission
	if err := s.db.WithContext(ctx).First(&submission, "id = ?", submissionID).Error; err != nil {
		return nil, nil, err
	}
	var items []domain.SubmissionItem
	if err := s.db.WithContext(ctx).Where("submission_id = ?", submission.ID).Find(&items).Error; err != nil {
		return nil, nil, err
	}
	return &submission, items, nil
}

func (s *SubmissionStore) UpdateGrades(ctx context.Context, submission *domain.AssignmentSubmission, items []domain.SubmissionItem) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.AssignmentSubmission{}).
			Where("id = ?", submission.ID).
			Updates(map[string]interface{}{
				"score":        submission.Score,
				"feedback":     submission.Feedback,
				"status":       submission.Status,
				"updated_at":   submission.UpdatedAt,
				"submitted_at": submission.SubmittedAt,
			}).Error; err != nil {
			return err
		}
		for _, item := range items {
			if err := tx.Model(&domain.SubmissionItem{}).
				Where("id = ? AND submission_id = ?", item.ID, submission.ID).
				Updates(map[string]interface{}{
					"score":  item.Score,
					"answer": item.Answer,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SubmissionStore) ListByStudentAndAssignments(ctx context.Context, studentID string, assignmentIDs []string) ([]domain.AssignmentSubmission, error) {
	if len(assignmentIDs) == 0 {
		return []domain.AssignmentSubmission{}, nil
	}

	var submissions []domain.AssignmentSubmission
	if err := s.db.WithContext(ctx).
		Where("student_id = ? AND assignment_id IN ?", studentID, assignmentIDs).
		Find(&submissions).Error; err != nil {
		return nil, err
	}
	return submissions, nil
}

func (s *SubmissionStore) StatsByAssignments(ctx context.Context, assignmentIDs []string) ([]repository.AssignmentSubmissionStats, error) {
	if len(assignmentIDs) == 0 {
		return []repository.AssignmentSubmissionStats{}, nil
	}

	type row struct {
		AssignmentID string
		Total        int64
		Submitted    int64
		Graded       int64
		Latest       *time.Time
		AvgScore     *float64
		MaxScore     *float64
		MinScore     *float64
		BucketLt60   int64
		Bucket60To70 int64
		Bucket70To80 int64
		Bucket80To90 int64
		BucketGte90  int64
	}

	var rows []row
	if err := s.db.WithContext(ctx).
		Model(&domain.AssignmentSubmission{}).
		Select("assignment_id as assignment_id",
			"COUNT(*) as total",
			"SUM(CASE WHEN status IN ('submitted','graded') THEN 1 ELSE 0 END) as submitted",
			"SUM(CASE WHEN status = 'graded' THEN 1 ELSE 0 END) as graded",
			"MAX(submitted_at) as latest",
			"AVG(score) as avg_score",
			"MAX(score) as max_score",
			"MIN(score) as min_score",
			"SUM(CASE WHEN score IS NOT NULL AND score < 60 THEN 1 ELSE 0 END) as bucket_lt_60",
			"SUM(CASE WHEN score IS NOT NULL AND score >= 60 AND score < 70 THEN 1 ELSE 0 END) as bucket_60_70",
			"SUM(CASE WHEN score IS NOT NULL AND score >= 70 AND score < 80 THEN 1 ELSE 0 END) as bucket_70_80",
			"SUM(CASE WHEN score IS NOT NULL AND score >= 80 AND score < 90 THEN 1 ELSE 0 END) as bucket_80_90",
			"SUM(CASE WHEN score IS NOT NULL AND score >= 90 THEN 1 ELSE 0 END) as bucket_gte_90").
		Where("assignment_id IN ?", assignmentIDs).
		Group("assignment_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	stats := make([]repository.AssignmentSubmissionStats, 0, len(rows))
	for _, r := range rows {
		stats = append(stats, repository.AssignmentSubmissionStats{
			AssignmentID:      r.AssignmentID,
			Total:             r.Total,
			Submitted:         r.Submitted,
			Graded:            r.Graded,
			LatestSubmittedAt: r.Latest,
			AverageScore:      r.AvgScore,
			MaxScore:          r.MaxScore,
			MinScore:          r.MinScore,
			ScoreDistribution: repository.AssignmentScoreDistribution{
				Below60:        r.BucketLt60,
				Between60And70: r.Bucket60To70,
				Between70And80: r.Bucket70To80,
				Between80And90: r.Bucket80To90,
				Above90:        r.BucketGte90,
			},
		})
	}
	return stats, nil
}

var _ repository.SubmissionRepository = (*SubmissionStore)(nil)
