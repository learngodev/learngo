package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

type listUsageSettingRepo struct {
	setting *domain.AIAgentSetting
	err     error
}

func (r *listUsageSettingRepo) GetBySchoolID(ctx context.Context, schoolID string) (*domain.AIAgentSetting, error) {
	return r.setting, r.err
}

func (r *listUsageSettingRepo) Upsert(context.Context, *domain.AIAgentSetting) error {
	return errors.New("not implemented")
}

type listUsageMessageRepo struct {
	rows     []repository.AIChatAccountUsage
	captured struct {
		schoolID string
		since    time.Time
		role     domain.Role
		limit    int
		offset   int
		sort     repository.AIChatUsageSort
	}
}

func (r *listUsageMessageRepo) Create(context.Context, *domain.AIChatMessage) error {
	return errors.New("not implemented")
}

func (r *listUsageMessageRepo) ListBySession(context.Context, string, int, time.Time) ([]domain.AIChatMessage, error) {
	return nil, errors.New("not implemented")
}

func (r *listUsageMessageRepo) CountUserMessagesSince(context.Context, string, time.Time) (int64, error) {
	return 0, errors.New("not implemented")
}

func (r *listUsageMessageRepo) UsageStatsByAccountSince(context.Context, string, time.Time) (repository.AIChatUsageStats, error) {
	return repository.AIChatUsageStats{}, errors.New("not implemented")
}

func (r *listUsageMessageRepo) UsageStatsBySchoolSince(ctx context.Context, schoolID string, since time.Time, role domain.Role, limit int, offset int, sort repository.AIChatUsageSort) ([]repository.AIChatAccountUsage, error) {
	r.captured.schoolID = schoolID
	r.captured.since = since
	r.captured.role = role
	r.captured.limit = limit
	r.captured.offset = offset
	r.captured.sort = sort
	return r.rows, nil
}

func (r *listUsageMessageRepo) UsageTotalsBySchoolSince(context.Context, string, time.Time, domain.Role) (repository.AIChatUsageTotals, error) {
	return repository.AIChatUsageTotals{}, errors.New("not implemented")
}

func (r *listUsageMessageRepo) UsageByRoleSince(context.Context, string, time.Time) (map[domain.Role]repository.AIChatUsageTotals, error) {
	return nil, errors.New("not implemented")
}

func (r *listUsageMessageRepo) UsageTimelineBySchool(context.Context, string, time.Time, time.Time, domain.Role) ([]repository.AIChatUsageTimelinePoint, error) {
	return nil, errors.New("not implemented")
}

type listUsageAccountRepo struct {
	accounts map[string]domain.Account
	received []string
}

func (r *listUsageAccountRepo) Create(context.Context, *domain.Account) error {
	return errors.New("not implemented")
}

func (r *listUsageAccountRepo) FindByIdentifier(context.Context, string, string) (*domain.Account, error) {
	return nil, errors.New("not implemented")
}

func (r *listUsageAccountRepo) FindByID(context.Context, string) (*domain.Account, error) {
	return nil, errors.New("not implemented")
}

func (r *listUsageAccountRepo) ListByIDs(ctx context.Context, ids []string) ([]domain.Account, error) {
	r.received = append([]string(nil), ids...)
	if len(ids) == 0 {
		return nil, nil
	}

	uniq := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	result := make([]domain.Account, 0, len(uniq))
	for _, id := range uniq {
		if acct, ok := r.accounts[id]; ok {
			result = append(result, acct)
		}
	}
	return result, nil
}

func (r *listUsageAccountRepo) ListByRole(context.Context, string, domain.Role, domain.AccountStatus, string, string, string, bool, bool, int, int, string) ([]domain.Account, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (r *listUsageAccountRepo) UpdateStatus(context.Context, string, string, domain.AccountStatus) error {
	return errors.New("not implemented")
}

func (r *listUsageAccountRepo) UpdatePasswordHash(context.Context, string, string) error {
	return errors.New("not implemented")
}

func (r *listUsageAccountRepo) Delete(context.Context, string, string) error {
	return errors.New("not implemented")
}

func TestListUsageSummariesUsesBatchAccounts(t *testing.T) {
	since := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)
	messageRepo := &listUsageMessageRepo{
		rows: []repository.AIChatAccountUsage{
			{AccountID: "acct-1", AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 3, AssistantMessages: 5, PromptTokens: 7, ResultTokens: 11}},
			{AccountID: "acct-2", AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 1, AssistantMessages: 2, PromptTokens: 3, ResultTokens: 4}},
		},
	}
	accountRepo := &listUsageAccountRepo{
		accounts: map[string]domain.Account{
			"acct-1": {ID: "acct-1", DisplayName: "Alice", Role: domain.RoleTeacher},
			"acct-2": {ID: "acct-2", DisplayName: "Bob", Role: domain.RoleStudent},
		},
	}
	settingRepo := &listUsageSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school-1", MaxDailyRequests: 5}}

	svc := &AIAssistantService{
		settings: settingRepo,
		messages: messageRepo,
		accounts: accountRepo,
		limiter:  newConcurrencyLimiter(),
	}

	summaries, err := svc.ListUsageSummaries(context.Background(), ListUsageSummariesInput{SchoolID: "school-1", Since: since})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	if accountRepo.received == nil || len(accountRepo.received) != 2 {
		t.Fatalf("expected ListByIDs to receive 2 ids, got %v", accountRepo.received)
	}

	first := summaries[0]
	if first.AccountName != "Alice" || first.Role != domain.RoleTeacher {
		t.Fatalf("unexpected first summary account info: %+v", first)
	}
	if first.TotalMessages != 8 || first.TotalTokens != 18 {
		t.Fatalf("unexpected aggregates for first summary: %+v", first)
	}
	if first.RemainingDailyRequests != 2 {
		t.Fatalf("expected remaining requests 2, got %d", first.RemainingDailyRequests)
	}

	second := summaries[1]
	if second.AccountName != "Bob" || second.Role != domain.RoleStudent {
		t.Fatalf("unexpected second summary account info: %+v", second)
	}
	if second.TotalMessages != 3 || second.TotalTokens != 7 {
		t.Fatalf("unexpected aggregates for second summary: %+v", second)
	}
	if second.RemainingDailyRequests != 4 {
		t.Fatalf("expected remaining requests 4, got %d", second.RemainingDailyRequests)
	}
}

func TestListUsageSummariesSkipsMissingAccounts(t *testing.T) {
	since := time.Date(2024, time.April, 2, 0, 0, 0, 0, time.UTC)
	messageRepo := &listUsageMessageRepo{
		rows: []repository.AIChatAccountUsage{
			{AccountID: "acct-1", AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 2, AssistantMessages: 2}},
			{AccountID: "acct-missing", AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 5, AssistantMessages: 5}},
		},
	}
	accountRepo := &listUsageAccountRepo{
		accounts: map[string]domain.Account{
			"acct-1": {ID: "acct-1", DisplayName: "Alice", Role: domain.RoleAdmin},
		},
	}
	settingRepo := &listUsageSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school-1", MaxDailyRequests: 10}}

	svc := &AIAssistantService{
		settings: settingRepo,
		messages: messageRepo,
		accounts: accountRepo,
		limiter:  newConcurrencyLimiter(),
	}

	summaries, err := svc.ListUsageSummaries(context.Background(), ListUsageSummariesInput{SchoolID: "school-1", Since: since, MinUserMessages: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	if summaries[0].AccountID != "acct-1" {
		t.Fatalf("expected summary for acct-1, got %s", summaries[0].AccountID)
	}
}

func TestListUsageSummariesDefaultPagination(t *testing.T) {
	messageRepo := &listUsageMessageRepo{
		rows: []repository.AIChatAccountUsage{},
	}
	accountRepo := &listUsageAccountRepo{accounts: map[string]domain.Account{}}
	settingRepo := &listUsageSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school-1"}}

	svc := &AIAssistantService{
		settings: settingRepo,
		messages: messageRepo,
		accounts: accountRepo,
		limiter:  newConcurrencyLimiter(),
	}

	if _, err := svc.ListUsageSummaries(context.Background(), ListUsageSummariesInput{SchoolID: "school-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if messageRepo.captured.limit != DefaultUsageSummaryPageSize {
		t.Fatalf("expected default limit %d, got %d", DefaultUsageSummaryPageSize, messageRepo.captured.limit)
	}
	if messageRepo.captured.offset != 0 {
		t.Fatalf("expected default offset 0, got %d", messageRepo.captured.offset)
	}
}

func TestListUsageSummariesCustomPagination(t *testing.T) {
	messageRepo := &listUsageMessageRepo{
		rows: []repository.AIChatAccountUsage{},
	}
	accountRepo := &listUsageAccountRepo{accounts: map[string]domain.Account{}}
	settingRepo := &listUsageSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school-1"}}

	svc := &AIAssistantService{
		settings: settingRepo,
		messages: messageRepo,
		accounts: accountRepo,
		limiter:  newConcurrencyLimiter(),
	}

	input := ListUsageSummariesInput{SchoolID: "school-1", Page: 3, PageSize: 15}
	if _, err := svc.ListUsageSummaries(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if messageRepo.captured.limit != 15 {
		t.Fatalf("expected limit 15, got %d", messageRepo.captured.limit)
	}
	expectedOffset := (3 - 1) * 15
	if messageRepo.captured.offset != expectedOffset {
		t.Fatalf("expected offset %d, got %d", expectedOffset, messageRepo.captured.offset)
	}
}

func TestListUsageSummariesHonorsSortConfig(t *testing.T) {
	messageRepo := &listUsageMessageRepo{
		rows: []repository.AIChatAccountUsage{
			{AccountID: "acct-1"},
		},
	}
	accountRepo := &listUsageAccountRepo{
		accounts: map[string]domain.Account{
			"acct-1": {ID: "acct-1", DisplayName: "Alice", Role: domain.RoleTeacher},
		},
	}
	settingRepo := &listUsageSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school-1"}}

	svc := &AIAssistantService{
		settings: settingRepo,
		messages: messageRepo,
		accounts: accountRepo,
		limiter:  newConcurrencyLimiter(),
	}

	input := ListUsageSummariesInput{
		SchoolID:      "school-1",
		SortField:     UsageSortFieldTotalTokens,
		SortDirection: UsageSortDirectionAsc,
	}

	if _, err := svc.ListUsageSummaries(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if messageRepo.captured.sort.Field != repository.AIChatUsageSortTotalTokens {
		t.Fatalf("expected sort field total_tokens, got %s", messageRepo.captured.sort.Field)
	}
	if messageRepo.captured.sort.Direction != repository.SortDirectionAsc {
		t.Fatalf("expected sort direction asc, got %s", messageRepo.captured.sort.Direction)
	}
}
