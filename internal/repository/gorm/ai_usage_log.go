package gormrepo

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

type AIUsageLogStore struct {
	db *gorm.DB
}

func NewAIUsageLogStore(db *gorm.DB) *AIUsageLogStore {
	return &AIUsageLogStore{db: db}
}

func (s *AIUsageLogStore) Create(ctx context.Context, log *domain.AIUsageLog) error {
	if log == nil {
		return errors.New("log required")
	}
	if log.ID == "" {
		return errors.New("log id required")
	}
	return s.db.WithContext(ctx).Create(log).Error
}

func (s *AIUsageLogStore) UsageByRoleSince(ctx context.Context, schoolID string, since time.Time) (map[domain.Role]repository.AIChatUsageTotals, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	query := s.db.WithContext(ctx).
		Model(&domain.AIUsageLog{}).
		Select(`
		role,
		COUNT(DISTINCT account_id) AS account_count,
		COUNT(*) AS total_interactions,
		SUM(prompt_tokens) AS prompt_tokens,
		SUM(result_tokens) AS result_tokens`).
		Where("school_id = ?", schoolID).
		Group("role")

	if !since.IsZero() {
		query = query.Where("created_at >= ?", since)
	}

	type row struct {
		Role              string
		AccountCount      sql.NullInt64
		TotalInteractions sql.NullInt64
		PromptTokens      sql.NullInt64
		ResultTokens      sql.NullInt64
	}

	var rows []row
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	convert := func(n sql.NullInt64) int64 {
		if !n.Valid {
			return 0
		}
		return n.Int64
	}

	result := make(map[domain.Role]repository.AIChatUsageTotals, len(rows))
	for _, r := range rows {
		role := domain.Role(strings.TrimSpace(r.Role))
		interactions := convert(r.TotalInteractions)

		result[role] = repository.AIChatUsageTotals{
			AccountCount: convert(r.AccountCount),
			AIChatUsageStats: repository.AIChatUsageStats{
				UserMessages:      interactions,
				AssistantMessages: interactions,
				PromptTokens:      convert(r.PromptTokens),
				ResultTokens:      convert(r.ResultTokens),
			},
		}
	}

	return result, nil
}

func (s *AIUsageLogStore) UsageTimelineBySchool(ctx context.Context, schoolID string, start, end time.Time, role domain.Role) ([]repository.AIChatUsageTimelinePoint, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	var dateTrunc string
	if s.db.Dialector.Name() == "sqlite" {
		dateTrunc = "strftime('%Y-%m-%d 00:00:00', created_at)"
	} else {
		dateTrunc = "DATE_TRUNC('day', created_at)"
	}

	query := s.db.WithContext(ctx).
		Model(&domain.AIUsageLog{}).
		Select(dateTrunc+` AS bucket,
		COUNT(DISTINCT account_id) AS account_count,
		COUNT(*) AS total_interactions,
		SUM(prompt_tokens) AS prompt_tokens,
		SUM(result_tokens) AS result_tokens`).
		Where("school_id = ?", schoolID).
		Group("bucket").
		Order("bucket ASC")

	if !start.IsZero() {
		query = query.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("created_at < ?", end)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	type row struct {
		Bucket            interface{}
		AccountCount      sql.NullInt64
		TotalInteractions sql.NullInt64
		PromptTokens      sql.NullInt64
		ResultTokens      sql.NullInt64
	}

	var rows []row
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	convert := func(n sql.NullInt64) int64 {
		if !n.Valid {
			return 0
		}
		return n.Int64
	}

	result := make([]repository.AIChatUsageTimelinePoint, 0, len(rows))
	for _, r := range rows {
		var bucketTime time.Time
		switch v := r.Bucket.(type) {
		case time.Time:
			bucketTime = v
		case string:
			// SQLite returns string
			parsed, err := time.Parse("2006-01-02 15:04:05", v)
			if err == nil {
				bucketTime = parsed
			}
		}

		interactions := convert(r.TotalInteractions)
		result = append(result, repository.AIChatUsageTimelinePoint{
			Bucket:       bucketTime,
			AccountCount: convert(r.AccountCount),
			AIChatUsageStats: repository.AIChatUsageStats{
				UserMessages:      interactions,
				AssistantMessages: interactions,
				PromptTokens:      convert(r.PromptTokens),
				ResultTokens:      convert(r.ResultTokens),
			},
		})
	}

	return result, nil
}
