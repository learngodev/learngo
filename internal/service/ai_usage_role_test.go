package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"gorm.io/gorm"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

type fakeSettingRepo struct {
	setting *domain.AIAgentSetting
	err     error
}

func (f *fakeSettingRepo) GetBySchoolID(ctx context.Context, schoolID string) (*domain.AIAgentSetting, error) {
	return f.setting, f.err
}

func (f *fakeSettingRepo) Upsert(ctx context.Context, setting *domain.AIAgentSetting) error {
	return errors.New("not implemented")
}

type fakeAuditRepo struct{}

func (f *fakeAuditRepo) Create(ctx context.Context, entry *domain.AIAgentSettingAudit) error {
	return errors.New("not implemented")
}

func (f *fakeAuditRepo) ListRecent(ctx context.Context, schoolID string, limit int) ([]domain.AIAgentSettingAudit, error) {
	return nil, errors.New("not implemented")
}

type fakeSessionRepo struct{}

func (f *fakeSessionRepo) Create(ctx context.Context, session *domain.AIChatSession) error {
	return errors.New("not implemented")
}
func (f *fakeSessionRepo) GetByID(ctx context.Context, sessionID string) (*domain.AIChatSession, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeSessionRepo) ListByAccount(ctx context.Context, accountID string, limit int) ([]domain.AIChatSession, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeSessionRepo) UpdateFields(ctx context.Context, sessionID string, updates map[string]any) error {
	return errors.New("not implemented")
}

func (f *fakeSessionRepo) Delete(ctx context.Context, sessionID string) error {
	return errors.New("not implemented")
}

type fakeAccountRepo struct{}

func (f *fakeAccountRepo) Create(ctx context.Context, account *domain.Account) error {
	return errors.New("not implemented")
}
func (f *fakeAccountRepo) FindByIdentifier(ctx context.Context, schoolID, identifier string) (*domain.Account, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeAccountRepo) FindByID(ctx context.Context, id string) (*domain.Account, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeAccountRepo) ListByIDs(ctx context.Context, ids []string) ([]domain.Account, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeAccountRepo) ListByRole(ctx context.Context, schoolID string, role domain.Role, status domain.AccountStatus, departmentID string, classID string, courseID string, onlyClassless bool, onlyDepartmentless bool, page int, size int, query string) ([]domain.Account, int64, error) {
	return nil, 0, errors.New("not implemented")
}
func (f *fakeAccountRepo) UpdateStatus(ctx context.Context, accountID, schoolID string, status domain.AccountStatus) error {
	return errors.New("not implemented")
}
func (f *fakeAccountRepo) UpdatePasswordHash(ctx context.Context, accountID string, passwordHash string) error {
	return errors.New("not implemented")
}
func (f *fakeAccountRepo) Delete(ctx context.Context, accountID, schoolID string) error {
	return errors.New("not implemented")
}

type fakeMessageRepo struct {
	totals   map[domain.Role]repository.AIChatUsageTotals
	err      error
	captured struct {
		schoolID string
		since    time.Time
	}
}

func (f *fakeMessageRepo) Create(ctx context.Context, message *domain.AIChatMessage) error {
	return errors.New("not implemented")
}
func (f *fakeMessageRepo) ListBySession(ctx context.Context, sessionID string, limit int, before time.Time) ([]domain.AIChatMessage, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeMessageRepo) CountUserMessagesSince(ctx context.Context, accountID string, since time.Time) (int64, error) {
	return 0, errors.New("not implemented")
}
func (f *fakeMessageRepo) UsageStatsByAccountSince(ctx context.Context, accountID string, since time.Time) (repository.AIChatUsageStats, error) {
	return repository.AIChatUsageStats{}, errors.New("not implemented")
}
func (f *fakeMessageRepo) UsageStatsBySchoolSince(ctx context.Context, schoolID string, since time.Time, role domain.Role, limit int, offset int, sort repository.AIChatUsageSort) ([]repository.AIChatAccountUsage, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeMessageRepo) UsageTotalsBySchoolSince(ctx context.Context, schoolID string, since time.Time, role domain.Role) (repository.AIChatUsageTotals, error) {
	return repository.AIChatUsageTotals{}, errors.New("not implemented")
}
func (f *fakeMessageRepo) UsageByRoleSince(ctx context.Context, schoolID string, since time.Time) (map[domain.Role]repository.AIChatUsageTotals, error) {
	f.captured.schoolID = schoolID
	f.captured.since = since
	return f.totals, f.err
}

func (f *fakeMessageRepo) UsageTimelineBySchool(context.Context, string, time.Time, time.Time, domain.Role) ([]repository.AIChatUsageTimelinePoint, error) {
	return nil, errors.New("not implemented")
}

func newTestService() *AIAssistantService {
	return &AIAssistantService{
		settings: &fakeSettingRepo{setting: &domain.AIAgentSetting{ID: "setting", SchoolID: "school"}},
		audits:   &fakeAuditRepo{},
		sessions: &fakeSessionRepo{},
		messages: &fakeMessageRepo{},
		accounts: &fakeAccountRepo{},
		limiter:  newConcurrencyLimiter(),
	}
}

func TestGetUsageRoleBreakdownRequiresSchoolID(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetUsageRoleBreakdown(context.Background(), ListUsageSummariesInput{})
	if err == nil || err.Error() != "school_id required" {
		t.Fatalf("expected school_id validation error, got %v", err)
	}
}

func TestGetUsageRoleBreakdownHandlesMissingSetting(t *testing.T) {
	svc := newTestService()
	svc.settings = &fakeSettingRepo{err: gorm.ErrRecordNotFound}

	_, err := svc.GetUsageRoleBreakdown(context.Background(), ListUsageSummariesInput{SchoolID: "school"})
	if !errors.Is(err, ErrAISettingNotConfigured) {
		t.Fatalf("expected ErrAISettingNotConfigured, got %v", err)
	}
}

func TestGetUsageRoleBreakdownReturnsAggregates(t *testing.T) {
	totals := map[domain.Role]repository.AIChatUsageTotals{
		domain.RoleTeacher: {AccountCount: 2, AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 4, AssistantMessages: 6, PromptTokens: 10, ResultTokens: 12}},
		domain.RoleStudent: {AccountCount: 1, AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 1, AssistantMessages: 3, PromptTokens: 5, ResultTokens: 7}},
	}

	repo := &fakeMessageRepo{totals: totals}

	svc := newTestService()
	svc.messages = repo

	summaries, err := svc.GetUsageRoleBreakdown(context.Background(), ListUsageSummariesInput{SchoolID: "school"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.captured.schoolID != "school" {
		t.Fatalf("expected schoolID to be captured, got %s", repo.captured.schoolID)
	}
	if repo.captured.since.IsZero() {
		t.Fatalf("expected since to be set to start of day")
	}

	if len(summaries) != len(usageRoleOrder) {
		t.Fatalf("expected %d summaries, got %d", len(usageRoleOrder), len(summaries))
	}

	expectedMessageShare := map[domain.Role]float64{
		domain.RoleTeacher: 10.0 / 14.0,
		domain.RoleStudent: 4.0 / 14.0,
	}
	expectedTokenShare := map[domain.Role]float64{
		domain.RoleTeacher: 22.0 / 34.0,
		domain.RoleStudent: 12.0 / 34.0,
	}

	const tolerance = 1e-9

	check := func(role domain.Role, summary AIChatUsageRoleSummary) {
		total := summary.UserMessages + summary.AssistantMessages
		if total != summary.TotalMessages {
			t.Fatalf("summary total mismatch for %s", role)
		}
		tokenTotal := summary.PromptTokens + summary.ResultTokens
		if tokenTotal != summary.TotalTokens {
			t.Fatalf("token total mismatch for %s", role)
		}

		if exp := expectedMessageShare[role]; math.Abs(summary.MessageShare-exp) > tolerance {
			t.Fatalf("unexpected message share for %s: got %f want %f", role, summary.MessageShare, exp)
		}
		if exp := expectedTokenShare[role]; math.Abs(summary.TokenShare-exp) > tolerance {
			t.Fatalf("unexpected token share for %s: got %f want %f", role, summary.TokenShare, exp)
		}

		if summary.AccountCount > 0 {
			avgMessages := float64(summary.TotalMessages) / float64(summary.AccountCount)
			if math.Abs(summary.AverageMessages-avgMessages) > tolerance {
				t.Fatalf("unexpected avg messages for %s: got %f want %f", role, summary.AverageMessages, avgMessages)
			}
			avgTokens := float64(summary.TotalTokens) / float64(summary.AccountCount)
			if math.Abs(summary.AverageTokens-avgTokens) > tolerance {
				t.Fatalf("unexpected avg tokens for %s: got %f want %f", role, summary.AverageTokens, avgTokens)
			}
		} else {
			if summary.AverageMessages != 0 || summary.AverageTokens != 0 {
				t.Fatalf("expected zero averages for %s", role)
			}
		}
	}

	for _, summary := range summaries {
		check(summary.Role, summary)
	}

	for idx, role := range usageRoleOrder {
		if summaries[idx].Role != role {
			t.Fatalf("expected role %s at index %d, got %s", role, idx, summaries[idx].Role)
		}
	}
}

func TestGetUsageRoleBreakdownEnsuresDefaultRoles(t *testing.T) {
	repo := &fakeMessageRepo{totals: map[domain.Role]repository.AIChatUsageTotals{}}
	svc := newTestService()
	svc.messages = repo

	summaries, err := svc.GetUsageRoleBreakdown(context.Background(), ListUsageSummariesInput{SchoolID: "school"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summaries) == 0 {
		t.Fatalf("expected default role summaries")
	}

	if len(summaries) != len(usageRoleOrder) {
		t.Fatalf("expected %d summaries, got %d", len(usageRoleOrder), len(summaries))
	}

	roleSet := make(map[domain.Role]AIChatUsageRoleSummary)
	for _, summary := range summaries {
		roleSet[summary.Role] = summary
	}

	for _, role := range usageRoleOrder {
		summary, ok := roleSet[role]
		if !ok {
			t.Fatalf("expected summary for role %s", role)
		}
		if summary.TotalMessages != 0 || summary.TotalTokens != 0 {
			t.Fatalf("expected zero usage for role %s", role)
		}
	}

	for idx, role := range usageRoleOrder {
		if summaries[idx].Role != role {
			t.Fatalf("expected role %s at index %d, got %s", role, idx, summaries[idx].Role)
		}
	}
}
