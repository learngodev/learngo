package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

type concurrencyLimiter struct {
	mu     sync.Mutex
	counts map[string]int
}

func newConcurrencyLimiter() *concurrencyLimiter {
	return &concurrencyLimiter{counts: make(map[string]int)}
}

func (l *concurrencyLimiter) TryAcquire(key string, max int) (bool, func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.counts[key]
	if current >= max {
		return false, nil
	}

	l.counts[key] = current + 1
	released := false

	release := func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if released {
			return
		}
		released = true

		remaining := l.counts[key] - 1
		if remaining <= 0 {
			delete(l.counts, key)
			return
		}
		l.counts[key] = remaining
	}

	return true, release
}

// AIAssistantService coordinates AI assistant configuration and usage flows.
type AIAssistantService struct {
	settings repository.AIAgentSettingRepository
	audits   repository.AIAgentSettingAuditRepository
	sessions repository.AIChatSessionRepository
	messages repository.AIChatMessageRepository
	accounts repository.AccountRepository
	model    AIChatModel
	limiter  *concurrencyLimiter
}

// NewAIAssistantService constructs the AI assistant service.
func NewAIAssistantService(settings repository.AIAgentSettingRepository, audits repository.AIAgentSettingAuditRepository, sessions repository.AIChatSessionRepository, messages repository.AIChatMessageRepository, accounts repository.AccountRepository, model AIChatModel) *AIAssistantService {
	return &AIAssistantService{
		settings: settings,
		audits:   audits,
		sessions: sessions,
		messages: messages,
		accounts: accounts,
		model:    model,
		limiter:  newConcurrencyLimiter(),
	}
}

var (
	// ErrAIAccountNotFound indicates the authenticated account record is missing.
	ErrAIAccountNotFound = errors.New("ai account not found")
	// ErrAISettingNotConfigured signals that the AI assistant setting is absent.
	ErrAISettingNotConfigured = errors.New("ai assistant not configured")
	// ErrAIChatSessionNotFound indicates the requested AI chat session was not located.
	ErrAIChatSessionNotFound = errors.New("ai chat session not found")
	// ErrAIChatSessionForbidden indicates the caller cannot access the requested session.
	ErrAIChatSessionForbidden = errors.New("ai chat session forbidden")
	// ErrAIChatMessageEmpty indicates the user message payload is blank.
	ErrAIChatMessageEmpty = errors.New("ai chat message empty")
	// ErrAIChatSessionLimitReached indicates the session has reached its turn limit.
	ErrAIChatSessionLimitReached = errors.New("ai chat session limit reached")
	// ErrAIChatDailyLimitExceeded indicates the account exceeded the daily quota.
	ErrAIChatDailyLimitExceeded = errors.New("ai chat daily limit exceeded")
	// ErrAIChatSessionClosed indicates the session is already closed and cannot accept new messages.
	ErrAIChatSessionClosed = errors.New("ai chat session closed")
	// ErrAIChatSessionTitleEmpty indicates the provided session title is empty.
	ErrAIChatSessionTitleEmpty = errors.New("ai chat session title empty")
	// ErrAIChatConcurrentLimitReached indicates the concurrent usage cap was hit.
	ErrAIChatConcurrentLimitReached = errors.New("ai chat concurrent limit reached")
)

// CreateAIChatSessionInput defines parameters for starting a new AI assistant session.
type CreateAIChatSessionInput struct {
	AccountID string
	Title     string
}

// SendAIChatMessageInput defines parameters for sending a message to the assistant.
type SendAIChatMessageInput struct {
	AccountID string
	SessionID string
	Content   string
}

// SendAIChatMessageResult contains persisted chat artefacts.
type SendAIChatMessageResult struct {
	Session          *domain.AIChatSession
	UserMessage      *domain.AIChatMessage
	AssistantMessage *domain.AIChatMessage
	ProviderError    error
	ProviderReason   ProviderErrorReason
}

// AIChatUsageSummary captures aggregate usage for analytics or quota display.
type AIChatUsageSummary struct {
	AccountID              string
	AccountName            string
	Role                   domain.Role
	SchoolID               string
	Since                  time.Time
	UserMessages           int64
	AssistantMessages      int64
	TotalMessages          int64
	PromptTokens           int64
	ResultTokens           int64
	TotalTokens            int64
	MaxDailyRequests       int
	RemainingDailyRequests int
}

// AIChatUsageAggregate captures overall usage metrics.
type AIChatUsageAggregate struct {
	SchoolID          string
	RoleFilter        domain.Role
	Since             time.Time
	ActiveAccounts    int
	UserMessages      int64
	AssistantMessages int64
	TotalMessages     int64
	PromptTokens      int64
	ResultTokens      int64
	TotalTokens       int64
	MaxDailyRequests  int
}

// ListUsageSummariesInput configures school-level usage aggregation.
type ListUsageSummariesInput struct {
	SchoolID         string
	Since            time.Time
	Role             domain.Role
	MinUserMessages  int64
	MinTotalMessages int64
	SortField        UsageSortField
	SortDirection    UsageSortDirection
	Page             int
	PageSize         int
}

// UsageSortField re-exports available usage sort columns.
type UsageSortField = repository.AIChatUsageSortField

const (
	UsageSortFieldUserMessages  UsageSortField = repository.AIChatUsageSortUserMessages
	UsageSortFieldTotalMessages UsageSortField = repository.AIChatUsageSortTotalMessages
	UsageSortFieldTotalTokens   UsageSortField = repository.AIChatUsageSortTotalTokens
)

// UsageSortDirection re-exports supported sort directions.
type UsageSortDirection = repository.SortDirection

const (
	UsageSortDirectionAsc  UsageSortDirection = repository.SortDirectionAsc
	UsageSortDirectionDesc UsageSortDirection = repository.SortDirectionDesc
)

const DefaultUsageSummaryPageSize = 20

// UpdateAIAgentSettingInput collects configuration updates from administrators.
type UpdateAIAgentSettingInput struct {
	SchoolID                string
	Provider                domain.AIProvider
	Model                   string
	APIKey                  string
	BaseURL                 string
	Temperature             float32
	TopP                    float32
	MaxOutputTokens         int
	MaxDailyRequests        int
	MaxConcurrentRequests   int
	MaxConversationMessages int
	SystemPrompt            string
	VisionEnabled           bool
	OperatorID              string
	OperatorName            string
}

// GetSetting returns the AI assistant setting for a given school (if any).
func (s *AIAssistantService) GetSetting(ctx context.Context, schoolID string) (*domain.AIAgentSetting, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	setting, err := s.settings.GetBySchoolID(ctx, schoolID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return setting, nil
}

// GetUsageSummary aggregates usage for the specified account and optional start time.
func (s *AIAssistantService) GetUsageSummary(ctx context.Context, accountID string, since time.Time) (*AIChatUsageSummary, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, errors.New("account_id required")
	}

	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIAccountNotFound
		}
		return nil, err
	}

	setting, err := s.settings.GetBySchoolID(ctx, account.SchoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAISettingNotConfigured
		}
		return nil, err
	}

	if since.IsZero() {
		now := time.Now()
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	stats, err := s.messages.UsageStatsByAccountSince(ctx, account.ID, since)
	if err != nil {
		return nil, err
	}

	totalMessages := stats.UserMessages + stats.AssistantMessages
	totalTokens := stats.PromptTokens + stats.ResultTokens
	remaining := setting.MaxDailyRequests
	if remaining > 0 {
		used := stats.UserMessages
		if used >= int64(setting.MaxDailyRequests) {
			remaining = 0
		} else {
			remaining = setting.MaxDailyRequests - int(used)
		}
	}

	name := strings.TrimSpace(account.DisplayName)
	if name == "" {
		name = account.ID
	}

	return &AIChatUsageSummary{
		AccountID:              account.ID,
		AccountName:            name,
		Role:                   account.Role,
		SchoolID:               account.SchoolID,
		Since:                  since,
		UserMessages:           stats.UserMessages,
		AssistantMessages:      stats.AssistantMessages,
		TotalMessages:          totalMessages,
		PromptTokens:           stats.PromptTokens,
		ResultTokens:           stats.ResultTokens,
		TotalTokens:            totalTokens,
		MaxDailyRequests:       setting.MaxDailyRequests,
		RemainingDailyRequests: remaining,
	}, nil
}

// GetUsageAggregate returns overall usage totals for a school.
func (s *AIAssistantService) GetUsageAggregate(ctx context.Context, input ListUsageSummariesInput) (*AIChatUsageAggregate, error) {
	schoolID := strings.TrimSpace(input.SchoolID)
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	setting, err := s.settings.GetBySchoolID(ctx, schoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAISettingNotConfigured
		}
		return nil, err
	}

	since := input.Since
	if since.IsZero() {
		now := time.Now()
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	totals, err := s.messages.UsageTotalsBySchoolSince(ctx, schoolID, since, input.Role)
	if err != nil {
		return nil, err
	}

	activeAccounts := int(totals.AccountCount)
	totalMessages := totals.UserMessages + totals.AssistantMessages
	totalTokens := totals.PromptTokens + totals.ResultTokens

	return &AIChatUsageAggregate{
		SchoolID:          schoolID,
		RoleFilter:        input.Role,
		Since:             since,
		ActiveAccounts:    activeAccounts,
		UserMessages:      totals.UserMessages,
		AssistantMessages: totals.AssistantMessages,
		TotalMessages:     totalMessages,
		PromptTokens:      totals.PromptTokens,
		ResultTokens:      totals.ResultTokens,
		TotalTokens:       totalTokens,
		MaxDailyRequests:  setting.MaxDailyRequests,
	}, nil
}

// ListUsageSummaries aggregates usage across accounts in a school.
func (s *AIAssistantService) ListUsageSummaries(ctx context.Context, input ListUsageSummariesInput) ([]AIChatUsageSummary, error) {
	schoolID := strings.TrimSpace(input.SchoolID)
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	setting, err := s.settings.GetBySchoolID(ctx, schoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAISettingNotConfigured
		}
		return nil, err
	}

	since := input.Since
	if since.IsZero() {
		now := time.Now()
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	limit := input.PageSize
	if limit <= 0 {
		limit = DefaultUsageSummaryPageSize
	}

	page := input.Page
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	sortField := input.SortField
	switch sortField {
	case UsageSortFieldUserMessages, UsageSortFieldTotalMessages, UsageSortFieldTotalTokens:
	default:
		sortField = UsageSortFieldUserMessages
	}

	sortDirection := input.SortDirection
	switch sortDirection {
	case UsageSortDirectionAsc, UsageSortDirectionDesc:
	default:
		sortDirection = UsageSortDirectionDesc
	}

	sortConfig := repository.AIChatUsageSort{Field: sortField, Direction: sortDirection}

	rows, err := s.messages.UsageStatsBySchoolSince(ctx, schoolID, since, input.Role, limit, offset, sortConfig)
	if err != nil {
		return nil, err
	}

	accountIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.AccountID) != "" {
			accountIDs = append(accountIDs, row.AccountID)
		}
	}

	accountsByID := make(map[string]domain.Account, len(accountIDs))
	if len(accountIDs) > 0 {
		accounts, err := s.accounts.ListByIDs(ctx, accountIDs)
		if err != nil {
			return nil, err
		}
		for i := range accounts {
			acct := accounts[i]
			if acct.ID == "" {
				continue
			}
			accountsByID[acct.ID] = acct
		}
	}

	summaries := make([]AIChatUsageSummary, 0, len(rows))
	for _, row := range rows {
		account, ok := accountsByID[row.AccountID]
		if !ok {
			continue
		}

		name := strings.TrimSpace(account.DisplayName)
		if name == "" {
			name = account.ID
		}

		totalMessages := row.UserMessages + row.AssistantMessages
		totalTokens := row.PromptTokens + row.ResultTokens

		if input.MinUserMessages > 0 && row.UserMessages < input.MinUserMessages {
			continue
		}
		if input.MinTotalMessages > 0 && totalMessages < input.MinTotalMessages {
			continue
		}

		remaining := setting.MaxDailyRequests
		if remaining > 0 {
			used := row.UserMessages
			if used >= int64(setting.MaxDailyRequests) {
				remaining = 0
			} else {
				remaining = setting.MaxDailyRequests - int(used)
			}
		}

		summaries = append(summaries, AIChatUsageSummary{
			AccountID:              account.ID,
			AccountName:            name,
			Role:                   account.Role,
			SchoolID:               schoolID,
			Since:                  since,
			UserMessages:           row.UserMessages,
			AssistantMessages:      row.AssistantMessages,
			TotalMessages:          totalMessages,
			PromptTokens:           row.PromptTokens,
			ResultTokens:           row.ResultTokens,
			TotalTokens:            totalTokens,
			MaxDailyRequests:       setting.MaxDailyRequests,
			RemainingDailyRequests: remaining,
		})
	}

	return summaries, nil
}

// ListAllUsageSummaries retrieves all usage summaries for the given filter regardless of page size.
func (s *AIAssistantService) ListAllUsageSummaries(ctx context.Context, input ListUsageSummariesInput) ([]AIChatUsageSummary, error) {
	batchSize := input.PageSize
	if batchSize <= 0 {
		batchSize = DefaultUsageSummaryPageSize
	}
	page := input.Page
	if page <= 0 {
		page = 1
	}

	all := make([]AIChatUsageSummary, 0)
	for {
		paged := input
		paged.Page = page
		paged.PageSize = batchSize
		chunk, err := s.ListUsageSummaries(ctx, paged)
		if err != nil {
			return nil, err
		}
		if len(chunk) == 0 {
			break
		}
		all = append(all, chunk...)
		if len(chunk) < batchSize {
			break
		}
		page++
	}

	return all, nil
}

// UpdateSetting creates or updates the AI assistant configuration.
func (s *AIAssistantService) UpdateSetting(ctx context.Context, input UpdateAIAgentSettingInput) (*domain.AIAgentSetting, error) {
	if input.SchoolID == "" {
		return nil, errors.New("school_id required")
	}
	if input.OperatorID == "" {
		return nil, errors.New("operator_id required")
	}
	if input.Provider == "" {
		return nil, errors.New("provider required")
	}

	switch input.Provider {
	case domain.AIProviderQwen, domain.AIProviderDeepSeek:
	default:
		return nil, errors.New("unsupported provider")
	}

	if input.Model == "" {
		return nil, errors.New("model required")
	}
	if input.Temperature < 0 || input.Temperature > 2 {
		return nil, errors.New("temperature must be between 0 and 2")
	}
	if input.TopP < 0 || input.TopP > 1 {
		return nil, errors.New("top_p must be between 0 and 1")
	}
	if input.MaxOutputTokens < 0 {
		return nil, errors.New("max_output_tokens must be >= 0")
	}
	if input.MaxDailyRequests < 0 {
		return nil, errors.New("max_daily_requests must be >= 0")
	}
	if input.MaxConcurrentRequests < 0 {
		return nil, errors.New("max_concurrent_requests must be >= 0")
	}
	if input.MaxConversationMessages < 0 {
		return nil, errors.New("max_conversation_messages must be >= 0")
	}

	existing, err := s.settings.GetBySchoolID(ctx, input.SchoolID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	var setting *domain.AIAgentSetting
	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		if input.APIKey == "" {
			return nil, errors.New("api_key required for initial configuration")
		}
		setting = &domain.AIAgentSetting{
			ID:       uuid.NewString(),
			SchoolID: input.SchoolID,
		}
	} else {
		setting = existing
	}

	setting.Provider = input.Provider
	setting.Model = input.Model
	if input.APIKey != "" {
		setting.APIKey = input.APIKey
	}
	setting.BaseURL = input.BaseURL
	setting.Temperature = input.Temperature
	setting.TopP = input.TopP
	setting.MaxOutputTokens = input.MaxOutputTokens
	setting.MaxDailyRequests = input.MaxDailyRequests
	setting.MaxConcurrentRequests = input.MaxConcurrentRequests
	setting.MaxConversationMessages = input.MaxConversationMessages
	setting.SystemPrompt = input.SystemPrompt
	setting.VisionEnabled = input.VisionEnabled
	setting.UpdatedBy = input.OperatorID

	operatorName := strings.TrimSpace(input.OperatorName)
	if operatorName == "" && s.accounts != nil {
		if account, err := s.accounts.FindByID(ctx, input.OperatorID); err == nil && account != nil {
			operatorName = account.DisplayName
		}
	}
	if operatorName == "" {
		operatorName = input.OperatorID
	}
	setting.UpdatedByName = operatorName
	setting.UpdatedAt = now

	if err := s.settings.Upsert(ctx, setting); err != nil {
		return nil, err
	}

	audit := &domain.AIAgentSettingAudit{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		OperatorID:   input.OperatorID,
		OperatorName: operatorName,
		Action:       "update_setting",
		Detail:       fmt.Sprintf("provider=%s, model=%s, vision=%t", setting.Provider, setting.Model, setting.VisionEnabled),
		CreatedAt:    now,
	}
	if err := s.audits.Create(ctx, audit); err != nil {
		return nil, err
	}

	return setting, nil
}

// ListSettingAudits returns recent audit entries for AI assistant configuration.
func (s *AIAssistantService) ListSettingAudits(ctx context.Context, schoolID string, limit int) ([]domain.AIAgentSettingAudit, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}
	if limit <= 0 {
		limit = 20
	}
	return s.audits.ListRecent(ctx, schoolID, limit)
}

// CreateSession starts a new AI chat session for the requesting account.
func (s *AIAssistantService) CreateSession(ctx context.Context, input CreateAIChatSessionInput) (*domain.AIChatSession, error) {
	if strings.TrimSpace(input.AccountID) == "" {
		return nil, errors.New("account_id required")
	}

	account, err := s.accounts.FindByID(ctx, input.AccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIAccountNotFound
		}
		return nil, err
	}

	if _, err := s.settings.GetBySchoolID(ctx, account.SchoolID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAISettingNotConfigured
		}
		return nil, err
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "AI Assistant"
	}

	now := time.Now()
	session := &domain.AIChatSession{
		ID:            uuid.NewString(),
		SchoolID:      account.SchoolID,
		AccountID:     account.ID,
		Role:          account.Role,
		Title:         title,
		LastMessageAt: now,
		MessageCount:  0,
		TokenCount:    0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// ListSessions returns recent sessions for the given account.
func (s *AIAssistantService) ListSessions(ctx context.Context, accountID string, limit int) ([]domain.AIChatSession, error) {
	if strings.TrimSpace(accountID) == "" {
		return nil, errors.New("account_id required")
	}
	if limit <= 0 {
		limit = 20
	}
	return s.sessions.ListByAccount(ctx, accountID, limit)
}

// ListSessionMessages returns chat messages for a session that belongs to the caller account.
func (s *AIAssistantService) ListSessionMessages(ctx context.Context, accountID, sessionID string, limit int, before time.Time) ([]domain.AIChatMessage, error) {
	if strings.TrimSpace(accountID) == "" {
		return nil, errors.New("account_id required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("session_id required")
	}

	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIChatSessionNotFound
		}
		return nil, err
	}
	if session.AccountID != accountID {
		return nil, ErrAIChatSessionForbidden
	}

	if limit <= 0 {
		limit = 20
	}

	messages, err := s.messages.ListBySession(ctx, sessionID, limit, before)
	if err != nil {
		return nil, err
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// SendMessage records a user utterance and enforces quota constraints.
func (s *AIAssistantService) SendMessage(ctx context.Context, input SendAIChatMessageInput) (*SendAIChatMessageResult, error) {
	accountID := strings.TrimSpace(input.AccountID)
	if accountID == "" {
		return nil, errors.New("account_id required")
	}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		return nil, errors.New("session_id required")
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return nil, ErrAIChatMessageEmpty
	}

	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIAccountNotFound
		}
		return nil, err
	}

	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIChatSessionNotFound
		}
		return nil, err
	}
	if session.AccountID != account.ID {
		return nil, ErrAIChatSessionForbidden
	}
	if session.ClosedAt != nil {
		return nil, ErrAIChatSessionClosed
	}

	setting, err := s.settings.GetBySchoolID(ctx, session.SchoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAISettingNotConfigured
		}
		return nil, err
	}

	now := time.Now()

	if setting.MaxDailyRequests > 0 {
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		count, err := s.messages.CountUserMessagesSince(ctx, account.ID, startOfDay)
		if err != nil {
			return nil, err
		}
		if int(count) >= setting.MaxDailyRequests {
			return nil, ErrAIChatDailyLimitExceeded
		}
	}

	if setting.MaxConversationMessages > 0 && session.MessageCount >= setting.MaxConversationMessages {
		return nil, ErrAIChatSessionLimitReached
	}

	var release func()
	if setting.MaxConcurrentRequests > 0 {
		var ok bool
		ok, release = s.limiter.TryAcquire(session.SchoolID, setting.MaxConcurrentRequests)
		if !ok {
			return nil, ErrAIChatConcurrentLimitReached
		}
		defer release()
	}

	userMessage := &domain.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: session.ID,
		Sender:    "user",
		Content:   content,
		CreatedAt: now,
	}

	if err := s.messages.Create(ctx, userMessage); err != nil {
		return nil, err
	}

	updates := map[string]any{
		"last_message_at": now,
		"updated_at":      now,
		"message_count":   session.MessageCount + 1,
	}
	if err := s.sessions.UpdateFields(ctx, session.ID, updates); err != nil {
		return nil, err
	}

	session.LastMessageAt = now
	session.UpdatedAt = now
	session.MessageCount++

	result := &SendAIChatMessageResult{
		Session:        session,
		UserMessage:    userMessage,
		ProviderReason: ProviderErrorReasonUnknown,
	}

	if s.model == nil {
		return result, nil
	}

	// Fetch recent history for context
	// Limit to last 10 messages to avoid token overflow, can be configurable
	historyLimit := 10
	historyMessages, err := s.messages.ListBySession(ctx, session.ID, historyLimit, now)
	if err != nil {
		// Log error but proceed without history? Or fail?
		// For now, let's proceed without history if fetch fails, or maybe just log it.
		// Since we don't have a logger here, we might just ignore or return error.
		// Let's return error to be safe.
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}

	// Reverse history to be chronological (oldest first)
	for i, j := 0, len(historyMessages)-1; i < j; i, j = i+1, j-1 {
		historyMessages[i], historyMessages[j] = historyMessages[j], historyMessages[i]
	}

	resp, err := s.model.GenerateResponse(ctx, AIChatModelRequest{
		Setting: setting,
		Session: session,
		Message: content,
		History: historyMessages,
	})
	if err != nil {
		providerErr := NormalizeProviderError(err)
		result.ProviderError = providerErr
		if providerErr != nil {
			result.ProviderReason = providerErr.Reason
		}
		return result, nil
	}

	reason := resp.Reason
	if reason == "" {
		reason = ProviderErrorReasonUnknown
	}
	result.ProviderReason = reason

	assistantAt := time.Now()
	assistantMessage := &domain.AIChatMessage{
		ID:           uuid.NewString(),
		SessionID:    session.ID,
		Sender:       "assistant",
		Content:      resp.Content,
		PromptTokens: resp.PromptTokens,
		ResultTokens: resp.ResultTokens,
		LatencyMS:    int(resp.Latency / time.Millisecond),
		CreatedAt:    assistantAt,
	}

	if err := s.messages.Create(ctx, assistantMessage); err != nil {
		result.ProviderError = &ProviderError{Reason: ProviderErrorReasonTransport, Err: err}
		return result, nil
	}

	session.LastMessageAt = assistantAt
	session.UpdatedAt = assistantAt
	session.MessageCount++
	session.TokenCount += resp.PromptTokens + resp.ResultTokens

	if err := s.sessions.UpdateFields(ctx, session.ID, map[string]any{
		"last_message_at": assistantAt,
		"updated_at":      assistantAt,
		"message_count":   session.MessageCount,
		"token_count":     session.TokenCount,
	}); err != nil {
		result.ProviderError = &ProviderError{Reason: ProviderErrorReasonTransport, Err: err}
		return result, nil
	}

	result.AssistantMessage = assistantMessage
	return result, nil
}

// CloseSession marks an AI chat session as closed for further interaction.
func (s *AIAssistantService) CloseSession(ctx context.Context, accountID, sessionID string) (*domain.AIChatSession, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, errors.New("account_id required")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errors.New("session_id required")
	}

	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIAccountNotFound
		}
		return nil, err
	}

	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIChatSessionNotFound
		}
		return nil, err
	}
	if session.AccountID != account.ID {
		return nil, ErrAIChatSessionForbidden
	}

	if session.ClosedAt != nil {
		return session, nil
	}

	now := time.Now()
	session.ClosedAt = &now
	session.UpdatedAt = now

	if err := s.sessions.UpdateFields(ctx, session.ID, map[string]any{
		"closed_at":  now,
		"updated_at": now,
	}); err != nil {
		return nil, err
	}

	return session, nil
}

// UpdateSessionTitle renames an AI chat session owned by the account.
func (s *AIAssistantService) UpdateSessionTitle(ctx context.Context, accountID, sessionID, title string) (*domain.AIChatSession, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, errors.New("account_id required")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errors.New("session_id required")
	}

	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIAccountNotFound
		}
		return nil, err
	}

	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAIChatSessionNotFound
		}
		return nil, err
	}
	if session.AccountID != account.ID {
		return nil, ErrAIChatSessionForbidden
	}

	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return nil, ErrAIChatSessionTitleEmpty
	}

	if session.Title == trimmed {
		return session, nil
	}

	now := time.Now()
	session.Title = trimmed
	session.UpdatedAt = now

	if err := s.sessions.UpdateFields(ctx, session.ID, map[string]any{
		"title":      trimmed,
		"updated_at": now,
	}); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *AIAssistantService) DeleteSession(ctx context.Context, accountID, sessionID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return errors.New("account_id required")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errors.New("session_id required")
	}

	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAIChatSessionNotFound
		}
		return err
	}
	if session.AccountID != accountID {
		return ErrAIChatSessionForbidden
	}

	return s.sessions.Delete(ctx, sessionID)
}
