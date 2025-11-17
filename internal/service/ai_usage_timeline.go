package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

// UsageTimelineInterval enumerates supported aggregation buckets.
type UsageTimelineInterval string

const (
	// UsageTimelineIntervalDay aggregates metrics per calendar day.
	UsageTimelineIntervalDay UsageTimelineInterval = "day"
)

const (
	// DefaultUsageTimelineDays defines the fallback window for timeline queries.
	DefaultUsageTimelineDays = 14
	// MaxUsageTimelineDays enforces an upper bound to guard expensive scans.
	MaxUsageTimelineDays = 90
)

var (
	// ErrUsageTimelineUnsupportedInterval signals an invalid aggregation interval.
	ErrUsageTimelineUnsupportedInterval = errors.New("usage timeline unsupported interval")
	// ErrUsageTimelineInvalidRange indicates the requested time range is invalid or too large.
	ErrUsageTimelineInvalidRange = errors.New("usage timeline invalid range")
)

// UsageTimelineInput configures time-series usage aggregation.
type UsageTimelineInput struct {
	SchoolID   string
	Role       domain.Role
	Start      time.Time
	End        time.Time
	Interval   UsageTimelineInterval
	WindowDays int
}

// AIChatUsageTimelinePoint captures usage totals for a single day.
type AIChatUsageTimelinePoint struct {
	Date              time.Time
	AccountCount      int
	UserMessages      int64
	AssistantMessages int64
	TotalMessages     int64
	PromptTokens      int64
	ResultTokens      int64
	TotalTokens       int64
	AverageMessages   float64
	AverageTokens     float64
}

// GetUsageTimeline returns per-day usage metrics for a school and optional role.
func (s *AIAssistantService) GetUsageTimeline(ctx context.Context, input UsageTimelineInput) ([]AIChatUsageTimelinePoint, error) {
	schoolID := strings.TrimSpace(input.SchoolID)
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	if s.settings == nil {
		return nil, errors.New("settings repository not configured")
	}
	if s.messages == nil {
		return nil, errors.New("message repository not configured")
	}

	if _, err := s.settings.GetBySchoolID(ctx, schoolID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAISettingNotConfigured
		}
		return nil, err
	}

	interval := input.Interval
	if interval == "" {
		interval = UsageTimelineIntervalDay
	}
	if interval != UsageTimelineIntervalDay {
		return nil, fmt.Errorf("%w: %s", ErrUsageTimelineUnsupportedInterval, interval)
	}

	end := input.End
	if end.IsZero() {
		end = time.Now()
	}
	end = end.UTC()
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)

	var startDay time.Time
	if !input.Start.IsZero() {
		start := input.Start.UTC()
		startDay = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	} else {
		window := input.WindowDays
		if window <= 0 {
			window = DefaultUsageTimelineDays
		}
		if window > MaxUsageTimelineDays {
			window = MaxUsageTimelineDays
		}
		// window defines inclusive day count.
		startDay = endDay.AddDate(0, 0, -(window - 1))
	}

	if endDay.Before(startDay) {
		return nil, fmt.Errorf("%w: end must not be before start", ErrUsageTimelineInvalidRange)
	}

	daySpan := int(endDay.Sub(startDay)/(24*time.Hour)) + 1
	if daySpan > MaxUsageTimelineDays {
		return nil, fmt.Errorf("%w: exceeds %d days", ErrUsageTimelineInvalidRange, MaxUsageTimelineDays)
	}

	queryEnd := endDay.Add(24 * time.Hour)

	rows, err := s.messages.UsageTimelineBySchool(ctx, schoolID, startDay, queryEnd, input.Role)
	if err != nil {
		return nil, err
	}

	keyed := make(map[time.Time]repository.AIChatUsageTimelinePoint, len(rows))
	for _, row := range rows {
		bucket := row.Bucket.UTC()
		bucket = time.Date(bucket.Year(), bucket.Month(), bucket.Day(), 0, 0, 0, 0, time.UTC)
		keyed[bucket] = row
	}

	timeline := make([]AIChatUsageTimelinePoint, 0, daySpan)
	for day := startDay; !day.After(endDay); day = day.Add(24 * time.Hour) {
		data := keyed[day]
		totalMessages := data.UserMessages + data.AssistantMessages
		totalTokens := data.PromptTokens + data.ResultTokens

		avgMessages := 0.0
		avgTokens := 0.0
		if data.AccountCount > 0 {
			avgMessages = float64(totalMessages) / float64(data.AccountCount)
			avgTokens = float64(totalTokens) / float64(data.AccountCount)
		}

		timeline = append(timeline, AIChatUsageTimelinePoint{
			Date:              day,
			AccountCount:      int(data.AccountCount),
			UserMessages:      data.UserMessages,
			AssistantMessages: data.AssistantMessages,
			TotalMessages:     totalMessages,
			PromptTokens:      data.PromptTokens,
			ResultTokens:      data.ResultTokens,
			TotalTokens:       totalTokens,
			AverageMessages:   avgMessages,
			AverageTokens:     avgTokens,
		})
	}

	return timeline, nil
}
