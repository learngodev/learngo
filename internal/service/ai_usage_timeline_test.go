package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

type timelineMessageRepo struct {
	rows     []repository.AIChatUsageTimelinePoint
	captured struct {
		schoolID string
		start    time.Time
		end      time.Time
		role     domain.Role
	}
}

func (r *timelineMessageRepo) Create(context.Context, *domain.AIChatMessage) error {
	return errors.New("not implemented")
}

func (r *timelineMessageRepo) ListBySession(context.Context, string, int, time.Time) ([]domain.AIChatMessage, error) {
	return nil, errors.New("not implemented")
}

func (r *timelineMessageRepo) CountUserMessagesSince(context.Context, string, time.Time) (int64, error) {
	return 0, errors.New("not implemented")
}

func (r *timelineMessageRepo) UsageStatsByAccountSince(context.Context, string, time.Time) (repository.AIChatUsageStats, error) {
	return repository.AIChatUsageStats{}, errors.New("not implemented")
}

func (r *timelineMessageRepo) UsageStatsBySchoolSince(context.Context, string, time.Time, domain.Role, int, int, repository.AIChatUsageSort) ([]repository.AIChatAccountUsage, error) {
	return nil, errors.New("not implemented")
}

func (r *timelineMessageRepo) UsageTotalsBySchoolSince(context.Context, string, time.Time, domain.Role) (repository.AIChatUsageTotals, error) {
	return repository.AIChatUsageTotals{}, errors.New("not implemented")
}

func (r *timelineMessageRepo) UsageByRoleSince(context.Context, string, time.Time) (map[domain.Role]repository.AIChatUsageTotals, error) {
	return nil, errors.New("not implemented")
}

func (r *timelineMessageRepo) UsageTimelineBySchool(ctx context.Context, schoolID string, start time.Time, end time.Time, role domain.Role) ([]repository.AIChatUsageTimelinePoint, error) {
	r.captured.schoolID = schoolID
	r.captured.start = start
	r.captured.end = end
	r.captured.role = role
	return r.rows, nil
}

func TestGetUsageTimelineFillsMissingDays(t *testing.T) {
	repo := &timelineMessageRepo{
		rows: []repository.AIChatUsageTimelinePoint{
			{
				Bucket:       time.Date(2024, time.May, 14, 8, 0, 0, 0, time.UTC),
				AccountCount: 2,
				AIChatUsageStats: repository.AIChatUsageStats{
					UserMessages:      3,
					AssistantMessages: 5,
					PromptTokens:      7,
					ResultTokens:      11,
				},
			},
			{
				Bucket:       time.Date(2024, time.May, 15, 10, 0, 0, 0, time.UTC),
				AccountCount: 1,
				AIChatUsageStats: repository.AIChatUsageStats{
					UserMessages:      2,
					AssistantMessages: 2,
					PromptTokens:      4,
					ResultTokens:      6,
				},
			},
		},
	}

	svc := &AIAssistantService{
		settings: &fakeSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school"}},
		messages: repo,
		limiter:  newConcurrencyLimiter(),
	}

	end := time.Date(2024, time.May, 15, 12, 0, 0, 0, time.UTC)
	timeline, err := svc.GetUsageTimeline(context.Background(), UsageTimelineInput{
		SchoolID:   "school",
		End:        end,
		WindowDays: 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.captured.schoolID != "school" {
		t.Fatalf("expected school capture, got %s", repo.captured.schoolID)
	}

	expectedStart := time.Date(2024, time.May, 13, 0, 0, 0, 0, time.UTC)
	if !repo.captured.start.Equal(expectedStart) {
		t.Fatalf("expected query start %s, got %s", expectedStart, repo.captured.start)
	}
	expectedEnd := time.Date(2024, time.May, 16, 0, 0, 0, 0, time.UTC)
	if !repo.captured.end.Equal(expectedEnd) {
		t.Fatalf("expected query end %s, got %s", expectedEnd, repo.captured.end)
	}

	if len(timeline) != 3 {
		t.Fatalf("expected 3 days, got %d", len(timeline))
	}

	if !timeline[0].Date.Equal(expectedStart) {
		t.Fatalf("expected first day %s, got %s", expectedStart, timeline[0].Date)
	}
	if timeline[0].TotalMessages != 0 || timeline[0].TotalTokens != 0 {
		t.Fatalf("expected zero metrics for empty day: %+v", timeline[0])
	}

	second := timeline[1]
	if second.TotalMessages != 8 || second.TotalTokens != 18 {
		t.Fatalf("unexpected aggregates for second day: %+v", second)
	}
	if second.AccountCount != 2 {
		t.Fatalf("unexpected account count: %d", second.AccountCount)
	}
	if second.AverageMessages != 4 || second.AverageTokens != 9 {
		t.Fatalf("unexpected averages: %+v", second)
	}
}

func TestGetUsageTimelineRejectsOversizedRange(t *testing.T) {
	repo := &timelineMessageRepo{}
	svc := &AIAssistantService{
		settings: &fakeSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school"}},
		messages: repo,
	}

	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 4, 0) // roughly 120 days

	_, err := svc.GetUsageTimeline(context.Background(), UsageTimelineInput{
		SchoolID: "school",
		Start:    start,
		End:      end,
	})
	if err == nil || !errors.Is(err, ErrUsageTimelineInvalidRange) {
		t.Fatalf("expected ErrUsageTimelineInvalidRange, got %v", err)
	}
}

func TestGetUsageTimelineHonorsExplicitRangeAndRole(t *testing.T) {
	repo := &timelineMessageRepo{}
	svc := &AIAssistantService{
		settings: &fakeSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school"}},
		messages: repo,
	}

	start := time.Date(2024, time.April, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2024, time.April, 10, 18, 0, 0, 0, time.UTC)

	if _, err := svc.GetUsageTimeline(context.Background(), UsageTimelineInput{
		SchoolID: "school",
		Role:     domain.RoleTeacher,
		Start:    start,
		End:      end,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedStart := time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2024, time.April, 11, 0, 0, 0, 0, time.UTC)

	if !repo.captured.start.Equal(expectedStart) {
		t.Fatalf("expected start %s, got %s", expectedStart, repo.captured.start)
	}
	if !repo.captured.end.Equal(expectedEnd) {
		t.Fatalf("expected end %s, got %s", expectedEnd, repo.captured.end)
	}
	if repo.captured.role != domain.RoleTeacher {
		t.Fatalf("expected role filter teacher, got %s", repo.captured.role)
	}
}
