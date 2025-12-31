package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
	"learn-go/internal/service"
	"learn-go/pkg/middleware"
)

func TestAIUsageAggregatePayloadSortsRoleBreakdown(t *testing.T) {
	gin.SetMode(gin.TestMode)

	breakdown := []service.AIChatUsageRoleSummary{
		{Role: domain.RoleAdmin, TotalMessages: 1},
		{Role: domain.RoleTeacher, TotalMessages: 3},
		{Role: domain.Role("guest"), TotalMessages: 2},
		{Role: domain.RoleStudent, TotalMessages: 4},
	}

	aggregate := service.AIChatUsageAggregate{}

	payload := aiUsageAggregatePayload(aggregate, breakdown)

	value, ok := payload["by_role"].([]gin.H)
	if !ok {
		t.Fatalf("expected by_role slice, got %T", payload["by_role"])
	}

	expectedOrder := []domain.Role{domain.RoleTeacher, domain.RoleStudent, domain.RoleAdmin, domain.Role("guest")}
	if len(value) != len(expectedOrder) {
		t.Fatalf("expected %d entries, got %d", len(expectedOrder), len(value))
	}

	for i, role := range expectedOrder {
		actualRole, ok := value[i]["role"].(string)
		if !ok {
			t.Fatalf("entry %d missing role string", i)
		}
		if actualRole != string(role) {
			t.Fatalf("expected role %s at index %d, got %s", role, i, actualRole)
		}
	}
}

func TestListAIUsageSummariesValidatesSortBy(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/usage?school_id=school&sort_by=invalid", nil)
	c.Request = req

	handler.ListAIUsageSummaries(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Message != "invalid sort_by" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestListAIUsageSummariesValidatesSortDir(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/usage?school_id=school&sort_dir=sideways", nil)
	c.Request = req

	handler.ListAIUsageSummaries(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Message != "invalid sort_dir" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestListAIUsageSummariesValidatesPageSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/usage?school_id=school&page_size=0", nil)
	c.Request = req

	handler.ListAIUsageSummaries(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Message != "invalid page_size" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestListAIUsageSummariesValidatesPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/usage?school_id=school&page=-2", nil)
	c.Request = req

	handler.ListAIUsageSummaries(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Message != "invalid page" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestExportAIUsageSummariesReturnsCSV(t *testing.T) {
	gin.SetMode(gin.TestMode)

	accountRepo := &stubAccountRepo{
		accounts: []domain.Account{
			{ID: "acct-1", DisplayName: "Alice", Role: domain.RoleTeacher},
			{ID: "acct-2", DisplayName: "Bob", Role: domain.RoleStudent},
		},
	}
	settingRepo := &stubSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school", MaxDailyRequests: 10}}
	messageRepo := &stubMessageRepo{
		usageStatsRows: []repository.AIChatAccountUsage{
			{AccountID: "acct-1", AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 5, AssistantMessages: 3, PromptTokens: 10, ResultTokens: 12}},
			{AccountID: "acct-2", AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 2, AssistantMessages: 1, PromptTokens: 4, ResultTokens: 6}},
		},
		usageStatsSet: true,
	}
	aiService := service.NewAIAssistantService(settingRepo, nil, nil, nil, messageRepo, accountRepo, nil, nil, nil, nil)
	handler := &Handler{ai: aiService}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/export?school_id=school&since=2024-05-01T00:00:00Z", nil)
	c.Request = req

	handler.ExportAIUsageSummaries(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("expected text/csv content type, got %s", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), ".csv") {
		t.Fatalf("expected csv attachment header, got %s", w.Header().Get("Content-Disposition"))
	}

	body := strings.TrimSpace(w.Body.String())
	lines := strings.Split(body, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 csv lines, got %d (%s)", len(lines), body)
	}
	if !strings.Contains(lines[1], "Alice") || !strings.Contains(lines[2], "Bob") {
		t.Fatalf("csv body missing account names: %v", lines)
	}
}

func TestGetAIUsageReportReturnsMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	accountRepo := &stubAccountRepo{
		accounts: []domain.Account{
			{ID: "acct-1", DisplayName: "Alice", Role: domain.RoleTeacher},
			{ID: "acct-2", DisplayName: "Bob", Role: domain.RoleStudent},
		},
	}
	settingRepo := &stubSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school", MaxDailyRequests: 20}}
	messageRepo := &stubMessageRepo{
		usageStatsRows: []repository.AIChatAccountUsage{
			{AccountID: "acct-1", AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 8, AssistantMessages: 4, PromptTokens: 12, ResultTokens: 10}},
			{AccountID: "acct-2", AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 2, AssistantMessages: 2, PromptTokens: 6, ResultTokens: 8}},
		},
		usageStatsSet: true,
		usageTotals: repository.AIChatUsageTotals{
			AccountCount: 2,
			AIChatUsageStats: repository.AIChatUsageStats{
				UserMessages:      10,
				AssistantMessages: 6,
				PromptTokens:      18,
				ResultTokens:      18,
			},
		},
		usageTotalsSet: true,
		usageRole: map[domain.Role]repository.AIChatUsageTotals{
			domain.RoleTeacher: {AccountCount: 1, AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 8, AssistantMessages: 4}},
			domain.RoleStudent: {AccountCount: 1, AIChatUsageStats: repository.AIChatUsageStats{UserMessages: 2, AssistantMessages: 2}},
		},
		usageRoleSet: true,
	}
	aiService := service.NewAIAssistantService(settingRepo, nil, nil, nil, messageRepo, accountRepo, nil, nil, nil, nil)
	handler := &Handler{ai: aiService}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/report?school_id=school", nil)
	c.Request = req

	handler.GetAIUsageReport(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing data payload: %+v", resp)
	}
	report, ok := data["report"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing report payload: %+v", data)
	}
	accounts, ok := report["accounts"].([]interface{})
	if !ok || len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %+v", report["accounts"])
	}
	metrics, ok := report["metrics"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing metrics: %+v", report)
	}
	if int(metrics["active_accounts"].(float64)) != 2 {
		t.Fatalf("unexpected active_accounts: %+v", metrics)
	}
	highlights, ok := report["highlights"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing highlights: %+v", report)
	}
	top, ok := highlights["top_account"].(map[string]interface{})
	if !ok || top["account_name"].(string) != "Alice" {
		t.Fatalf("unexpected top account highlight: %+v", highlights)
	}
	overview, ok := report["overview"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing overview: %+v", report)
	}
	if int(overview["total_messages"].(float64)) != 16 {
		t.Fatalf("unexpected total_messages in overview: %+v", overview)
	}
}

func TestGetAIUsageTimelineRequiresSchoolID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/timeline", nil)
	c.Request = req

	handler.GetAIUsageTimeline(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error.Message != "school_id required" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestGetAIUsageTimelineValidatesWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/timeline?school_id=school&window_days=abc", nil)
	c.Request = req

	handler.GetAIUsageTimeline(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error.Message != "invalid window_days" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestGetAIUsageTimelineReturnsPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rows := []repository.AIChatUsageTimelinePoint{
		{
			Bucket:       time.Date(2024, time.May, 9, 8, 0, 0, 0, time.UTC),
			AccountCount: 2,
			AIChatUsageStats: repository.AIChatUsageStats{
				UserMessages:      3,
				AssistantMessages: 5,
				PromptTokens:      7,
				ResultTokens:      11,
			},
		},
		{
			Bucket:       time.Date(2024, time.May, 10, 8, 0, 0, 0, time.UTC),
			AccountCount: 1,
			AIChatUsageStats: repository.AIChatUsageStats{
				UserMessages:      1,
				AssistantMessages: 1,
				PromptTokens:      2,
				ResultTokens:      2,
			},
		},
	}

	messageRepo := &stubMessageRepo{timelineRows: rows}
	settingRepo := &stubSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school"}}
	aiService := service.NewAIAssistantService(settingRepo, nil, nil, nil, messageRepo, nil, nil, nil, nil, nil)
	handler := &Handler{ai: aiService}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/timeline?school_id=school&end=2024-05-10T00:00:00Z&window_days=2", nil)
	c.Request = req

	handler.GetAIUsageTimeline(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp timelineResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success true")
	}
	if resp.Data.Interval != string(service.UsageTimelineIntervalDay) {
		t.Fatalf("unexpected interval %s", resp.Data.Interval)
	}
	if len(resp.Data.Timeline) != 2 {
		t.Fatalf("expected 2 timeline points, got %d", len(resp.Data.Timeline))
	}
	if resp.Data.Timeline[0].TotalMessages != 8 {
		t.Fatalf("unexpected first total messages: %+v", resp.Data.Timeline[0])
	}
	if resp.Data.Timeline[1].AccountCount != 1 {
		t.Fatalf("unexpected account count in second point")
	}
	if resp.Data.Start.IsZero() || resp.Data.End.IsZero() {
		t.Fatalf("expected start/end to be populated")
	}
}

func TestGetAIUsageTimelineReportsInvalidRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	messageRepo := &stubMessageRepo{}
	settingRepo := &stubSettingRepo{setting: &domain.AIAgentSetting{SchoolID: "school"}}
	aiService := service.NewAIAssistantService(settingRepo, nil, nil, nil, messageRepo, nil, nil, nil, nil, nil)
	handler := &Handler{ai: aiService}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/timeline?school_id=school&start=2024-01-01T00:00:00Z&end=2024-05-01T00:00:00Z", nil)
	c.Request = req

	handler.GetAIUsageTimeline(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error.Message != "invalid timeline request" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestGetAIUsageSummaryRequiresAccountContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	c.Request = req

	handler.GetAIUsageSummary(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Success {
		t.Fatal("expected success to be false")
	}
	if resp.Error.Message != "missing account context" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestGetAIUsageSummaryRejectsInvalidSince(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &Handler{}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	q := req.URL.Query()
	q.Set("since", "not-a-time")
	req.URL.RawQuery = q.Encode()
	c.Request = req
	c.Set(middleware.ContextAccountID, "acct-1")

	handler.GetAIUsageSummary(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Success {
		t.Fatal("expected success to be false")
	}
	if resp.Error.Message != "invalid since timestamp" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestGetAIUsageSummaryReturnsPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedSince := time.Date(2024, time.March, 10, 8, 0, 0, 0, time.UTC)

	accountRepo := &stubAccountRepo{
		account: &domain.Account{
			ID:          "acct-1",
			SchoolID:    "school-1",
			Role:        domain.RoleTeacher,
			DisplayName: "Alice",
		},
	}
	settingRepo := &stubSettingRepo{
		setting: &domain.AIAgentSetting{
			SchoolID:         "school-1",
			MaxDailyRequests: 10,
		},
	}
	messageRepo := &stubMessageRepo{
		stats: repository.AIChatUsageStats{
			UserMessages:      3,
			AssistantMessages: 7,
			PromptTokens:      5,
			ResultTokens:      11,
		},
	}

	aiService := service.NewAIAssistantService(settingRepo, nil, nil, nil, messageRepo, accountRepo, nil, nil, nil, nil)
	handler := &Handler{ai: aiService}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	q := req.URL.Query()
	q.Set("since", expectedSince.Format(time.RFC3339))
	req.URL.RawQuery = q.Encode()
	c.Request = req
	c.Set(middleware.ContextAccountID, "acct-1")

	handler.GetAIUsageSummary(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp usageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Fatal("expected success to be true")
	}

	usage := resp.Data.Usage
	if usage.AccountID != "acct-1" {
		t.Fatalf("unexpected account_id: %s", usage.AccountID)
	}
	if usage.AccountName != "Alice" {
		t.Fatalf("unexpected account_name: %s", usage.AccountName)
	}
	if usage.Role != string(domain.RoleTeacher) {
		t.Fatalf("unexpected role: %s", usage.Role)
	}
	if !usage.Since.Equal(expectedSince) {
		t.Fatalf("expected since %s, got %s", expectedSince, usage.Since)
	}
	if usage.TotalMessages != 10 {
		t.Fatalf("unexpected total_messages: %d", usage.TotalMessages)
	}
	if usage.TotalTokens != 16 {
		t.Fatalf("unexpected total_tokens: %d", usage.TotalTokens)
	}
	if usage.RemainingDailyRequests != 7 {
		t.Fatalf("unexpected remaining_daily_requests: %d", usage.RemainingDailyRequests)
	}

	if accountRepo.lastID != "acct-1" {
		t.Fatalf("expected account repo to be called with acct-1, got %s", accountRepo.lastID)
	}
	if messageRepo.lastAccountID != "acct-1" {
		t.Fatalf("expected message repo to be called with acct-1, got %s", messageRepo.lastAccountID)
	}
	if !messageRepo.lastSince.Equal(expectedSince) {
		t.Fatalf("expected message repo to receive since %s, got %s", expectedSince, messageRepo.lastSince)
	}
	if settingRepo.lastSchoolID != "school-1" {
		t.Fatalf("expected setting repo to be called with school-1, got %s", settingRepo.lastSchoolID)
	}
}

func TestGetAIUsageSummaryHandlesAccountMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	accountRepo := &stubAccountRepo{err: gorm.ErrRecordNotFound}
	settingRepo := &stubSettingRepo{}
	messageRepo := &stubMessageRepo{}

	aiService := service.NewAIAssistantService(settingRepo, nil, nil, nil, messageRepo, accountRepo, nil, nil, nil, nil)
	handler := &Handler{ai: aiService}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	c.Request = req
	c.Set(middleware.ContextAccountID, "acct-1")

	handler.GetAIUsageSummary(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Message != "account not found" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestGetAIUsageSummaryHandlesSettingMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	accountRepo := &stubAccountRepo{
		account: &domain.Account{ID: "acct-1", SchoolID: "school-1"},
	}
	settingRepo := &stubSettingRepo{err: gorm.ErrRecordNotFound}
	messageRepo := &stubMessageRepo{}

	aiService := service.NewAIAssistantService(settingRepo, nil, nil, nil, messageRepo, accountRepo, nil, nil, nil, nil)
	handler := &Handler{ai: aiService}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	c.Request = req
	c.Set(middleware.ContextAccountID, "acct-1")

	handler.GetAIUsageSummary(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Message != "ai assistant not configured" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

func TestGetAIUsageSummaryHandlesInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	accountRepo := &stubAccountRepo{
		account: &domain.Account{ID: "acct-1", SchoolID: "school-1"},
	}
	settingRepo := &stubSettingRepo{
		setting: &domain.AIAgentSetting{SchoolID: "school-1"},
	}
	messageRepo := &stubMessageRepo{err: errors.New("boom")}

	aiService := service.NewAIAssistantService(settingRepo, nil, nil, nil, messageRepo, accountRepo, nil, nil, nil, nil)
	handler := &Handler{ai: aiService}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	c.Request = req
	c.Set(middleware.ContextAccountID, "acct-1")

	handler.GetAIUsageSummary(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error.Message != "unable to fetch ai usage" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}

type usageResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Usage struct {
			AccountID              string    `json:"account_id"`
			AccountName            string    `json:"account_name"`
			Role                   string    `json:"role"`
			SchoolID               string    `json:"school_id"`
			Since                  time.Time `json:"since"`
			UserMessages           int64     `json:"user_messages"`
			AssistantMessages      int64     `json:"assistant_messages"`
			TotalMessages          int64     `json:"total_messages"`
			PromptTokens           int64     `json:"prompt_tokens"`
			ResultTokens           int64     `json:"result_tokens"`
			TotalTokens            int64     `json:"total_tokens"`
			MaxDailyRequests       int       `json:"max_daily_requests"`
			RemainingDailyRequests int       `json:"remaining_daily_requests"`
		} `json:"usage"`
	} `json:"data"`
}

type errorResponse struct {
	Success bool `json:"success"`
	Error   struct {
		Message string      `json:"message"`
		Details interface{} `json:"details"`
	} `json:"error"`
}

type timelineResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Timeline []struct {
			Date          time.Time `json:"date"`
			AccountCount  int       `json:"account_count"`
			TotalMessages int64     `json:"total_messages"`
			TotalTokens   int64     `json:"total_tokens"`
		} `json:"timeline"`
		Start    time.Time `json:"start"`
		End      time.Time `json:"end"`
		Interval string    `json:"interval"`
	} `json:"data"`
}

type stubAccountRepo struct {
	account     *domain.Account
	err         error
	lastID      string
	accounts    []domain.Account
	listErr     error
	lastListIDs []string
}

func (s *stubAccountRepo) Create(context.Context, *domain.Account) error {
	panic("unexpected call")
}

func (s *stubAccountRepo) FindByIdentifier(context.Context, string, string) (*domain.Account, error) {
	panic("unexpected call")
}

func (s *stubAccountRepo) FindByID(ctx context.Context, id string) (*domain.Account, error) {
	s.lastID = id
	if s.err != nil {
		return nil, s.err
	}
	return s.account, nil
}

func (s *stubAccountRepo) ListByIDs(ctx context.Context, ids []string) ([]domain.Account, error) {
	s.lastListIDs = append([]string(nil), ids...)
	if s.listErr != nil {
		return nil, s.listErr
	}
	if len(s.accounts) == 0 {
		return nil, nil
	}
	lookup := make(map[string]domain.Account, len(s.accounts))
	for _, acct := range s.accounts {
		lookup[acct.ID] = acct
	}
	result := make([]domain.Account, 0, len(ids))
	for _, id := range ids {
		if acct, ok := lookup[id]; ok {
			result = append(result, acct)
		}
	}
	return result, nil
}

func (s *stubAccountRepo) ListByRole(context.Context, string, domain.Role, domain.AccountStatus, string, string, string, bool, bool, int, int, string) ([]domain.Account, int64, error) {
	panic("unexpected call")
}

func (s *stubAccountRepo) UpdateStatus(context.Context, string, string, domain.AccountStatus) error {
	panic("unexpected call")
}

func (s *stubAccountRepo) UpdatePasswordHash(context.Context, string, string) error {
	panic("unexpected call")
}

func (s *stubAccountRepo) Update(ctx context.Context, account *domain.Account) error {
	panic("unexpected call")
}

func (s *stubAccountRepo) Delete(context.Context, string, string) error {
	panic("unexpected call")
}

type stubSettingRepo struct {
	setting      *domain.AIAgentSetting
	err          error
	lastSchoolID string
}

func (s *stubSettingRepo) GetBySchoolID(ctx context.Context, schoolID string) (*domain.AIAgentSetting, error) {
	s.lastSchoolID = schoolID
	if s.err != nil {
		return nil, s.err
	}
	return s.setting, nil
}

func (s *stubSettingRepo) Upsert(context.Context, *domain.AIAgentSetting) error {
	panic("unexpected call")
}

type stubMessageRepo struct {
	stats         repository.AIChatUsageStats
	err           error
	lastAccountID string
	lastSince     time.Time
	timelineRows  []repository.AIChatUsageTimelinePoint
	timelineErr   error
	lastTimeline  struct {
		schoolID string
		start    time.Time
		end      time.Time
		role     domain.Role
	}
	usageStatsRows []repository.AIChatAccountUsage
	usageStatsErr  error
	usageStatsSet  bool
	usageTotals    repository.AIChatUsageTotals
	usageTotalsErr error
	usageTotalsSet bool
	usageRole      map[domain.Role]repository.AIChatUsageTotals
	usageRoleErr   error
	usageRoleSet   bool
	lastStatsQuery struct {
		schoolID string
		since    time.Time
		role     domain.Role
		limit    int
		offset   int
		sort     repository.AIChatUsageSort
	}
}

func (s *stubMessageRepo) Create(context.Context, *domain.AIChatMessage) error {
	panic("unexpected call")
}

func (s *stubMessageRepo) ListBySession(context.Context, string, int, time.Time) ([]domain.AIChatMessage, error) {
	panic("unexpected call")
}

func (s *stubMessageRepo) CountUserMessagesSince(context.Context, string, time.Time) (int64, error) {
	panic("unexpected call")
}

func (s *stubMessageRepo) UsageStatsByAccountSince(ctx context.Context, accountID string, since time.Time) (repository.AIChatUsageStats, error) {
	s.lastAccountID = accountID
	s.lastSince = since
	if s.err != nil {
		return repository.AIChatUsageStats{}, s.err
	}
	return s.stats, nil
}

func (s *stubMessageRepo) UsageStatsBySchoolSince(ctx context.Context, schoolID string, since time.Time, role domain.Role, limit int, offset int, sort repository.AIChatUsageSort) ([]repository.AIChatAccountUsage, error) {
	if !s.usageStatsSet && s.usageStatsErr == nil {
		panic("unexpected call")
	}
	s.lastStatsQuery.schoolID = schoolID
	s.lastStatsQuery.since = since
	s.lastStatsQuery.role = role
	s.lastStatsQuery.limit = limit
	s.lastStatsQuery.offset = offset
	s.lastStatsQuery.sort = sort
	if s.usageStatsErr != nil {
		return nil, s.usageStatsErr
	}
	if limit <= 0 {
		limit = len(s.usageStatsRows)
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(s.usageStatsRows) {
		return []repository.AIChatAccountUsage{}, nil
	}
	end := offset + limit
	if end > len(s.usageStatsRows) {
		end = len(s.usageStatsRows)
	}
	return s.usageStatsRows[offset:end], nil
}

func (s *stubMessageRepo) UsageTotalsBySchoolSince(ctx context.Context, schoolID string, since time.Time, role domain.Role) (repository.AIChatUsageTotals, error) {
	if !s.usageTotalsSet && s.usageTotalsErr == nil {
		panic("unexpected call")
	}
	if s.usageTotalsErr != nil {
		return repository.AIChatUsageTotals{}, s.usageTotalsErr
	}
	return s.usageTotals, nil
}

func (s *stubMessageRepo) UsageByRoleSince(ctx context.Context, schoolID string, since time.Time) (map[domain.Role]repository.AIChatUsageTotals, error) {
	if !s.usageRoleSet && s.usageRoleErr == nil {
		panic("unexpected call")
	}
	if s.usageRoleErr != nil {
		return nil, s.usageRoleErr
	}
	return s.usageRole, nil
}

func (s *stubMessageRepo) UsageTimelineBySchool(ctx context.Context, schoolID string, start time.Time, end time.Time, role domain.Role) ([]repository.AIChatUsageTimelinePoint, error) {
	if s.timelineRows == nil && s.timelineErr == nil {
		panic("unexpected call")
	}
	s.lastTimeline.schoolID = schoolID
	s.lastTimeline.start = start
	s.lastTimeline.end = end
	s.lastTimeline.role = role
	return s.timelineRows, s.timelineErr
}
