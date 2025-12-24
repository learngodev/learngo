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

// AIAgentSettingStore implements repository.AIAgentSettingRepository.
type AIAgentSettingStore struct {
	db *gorm.DB
}

// NewAIAgentSettingStore constructs an AIAgentSettingStore.
func NewAIAgentSettingStore(db *gorm.DB) *AIAgentSettingStore {
	return &AIAgentSettingStore{db: db}
}

func (s *AIAgentSettingStore) GetBySchoolID(ctx context.Context, schoolID string) (*domain.AIAgentSetting, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	var setting domain.AIAgentSetting
	if err := s.db.WithContext(ctx).Where("school_id = ?", schoolID).First(&setting).Error; err != nil {
		return nil, err
	}
	return &setting, nil
}

func (s *AIAgentSettingStore) Upsert(ctx context.Context, setting *domain.AIAgentSetting) error {
	if setting == nil {
		return errors.New("setting required")
	}
	if setting.ID == "" {
		return errors.New("setting id required")
	}

	var existing domain.AIAgentSetting
	err := s.db.WithContext(ctx).First(&existing, "id = ?", setting.ID).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return s.db.WithContext(ctx).Create(setting).Error
	case err != nil:
		return err
	default:
		updates := map[string]any{
			"provider":                  setting.Provider,
			"model":                     setting.Model,
			"api_key":                   setting.APIKey,
			"base_url":                  setting.BaseURL,
			"temperature":               setting.Temperature,
			"top_p":                     setting.TopP,
			"max_output_tokens":         setting.MaxOutputTokens,
			"max_daily_requests":        setting.MaxDailyRequests,
			"max_concurrent_requests":   setting.MaxConcurrentRequests,
			"max_conversation_messages": setting.MaxConversationMessages,
			"system_prompt":             setting.SystemPrompt,
			"vision_enabled":            setting.VisionEnabled,
			"updated_by":                setting.UpdatedBy,
			"updated_by_name":           setting.UpdatedByName,
			"updated_at":                setting.UpdatedAt,
		}

		return s.db.WithContext(ctx).
			Model(&domain.AIAgentSetting{}).
			Where("id = ?", setting.ID).
			Updates(updates).Error
	}
}

var _ repository.AIAgentSettingRepository = (*AIAgentSettingStore)(nil)

// AIAgentSettingAuditStore implements repository.AIAgentSettingAuditRepository.
type AIAgentSettingAuditStore struct {
	db *gorm.DB
}

// NewAIAgentSettingAuditStore constructs an audit store.
func NewAIAgentSettingAuditStore(db *gorm.DB) *AIAgentSettingAuditStore {
	return &AIAgentSettingAuditStore{db: db}
}

func (s *AIAgentSettingAuditStore) Create(ctx context.Context, entry *domain.AIAgentSettingAudit) error {
	if entry == nil {
		return errors.New("entry required")
	}
	if entry.ID == "" {
		return errors.New("entry id required")
	}
	return s.db.WithContext(ctx).Create(entry).Error
}

func (s *AIAgentSettingAuditStore) ListRecent(ctx context.Context, schoolID string, limit int) ([]domain.AIAgentSettingAudit, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	if limit <= 0 {
		limit = 20
	}

	var entries []domain.AIAgentSettingAudit
	query := s.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("created_at DESC").
		Limit(limit)
	if err := query.Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

var _ repository.AIAgentSettingAuditRepository = (*AIAgentSettingAuditStore)(nil)

// AIChatSessionStore implements repository.AIChatSessionRepository.
type AIChatSessionStore struct {
	db *gorm.DB
}

// NewAIChatSessionStore constructs a chat session store.
func NewAIChatSessionStore(db *gorm.DB) *AIChatSessionStore {
	return &AIChatSessionStore{db: db}
}

func (s *AIChatSessionStore) Create(ctx context.Context, session *domain.AIChatSession) error {
	if session == nil {
		return errors.New("session required")
	}
	if session.ID == "" {
		return errors.New("session id required")
	}
	return s.db.WithContext(ctx).Create(session).Error
}

func (s *AIChatSessionStore) GetByID(ctx context.Context, sessionID string) (*domain.AIChatSession, error) {
	if sessionID == "" {
		return nil, errors.New("session_id required")
	}

	var session domain.AIChatSession
	if err := s.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *AIChatSessionStore) ListByAccount(ctx context.Context, accountID string, limit int) ([]domain.AIChatSession, error) {
	if accountID == "" {
		return nil, errors.New("account_id required")
	}

	if limit <= 0 {
		limit = 20
	}

	var sessions []domain.AIChatSession
	query := s.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Order("last_message_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *AIChatSessionStore) UpdateFields(ctx context.Context, sessionID string, updates map[string]any) error {
	if sessionID == "" {
		return errors.New("session_id required")
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Model(&domain.AIChatSession{}).
		Where("id = ?", sessionID).
		Updates(updates).Error
}

func (s *AIChatSessionStore) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return errors.New("session_id required")
	}
	return s.db.WithContext(ctx).Delete(&domain.AIChatSession{}, "id = ?", sessionID).Error
}

var _ repository.AIChatSessionRepository = (*AIChatSessionStore)(nil)

var _ repository.AIChatSessionRepository = (*AIChatSessionStore)(nil)

// AIChatMessageStore implements repository.AIChatMessageRepository.
type AIChatMessageStore struct {
	db *gorm.DB
}

// NewAIChatMessageStore constructs a chat message store.
func NewAIChatMessageStore(db *gorm.DB) *AIChatMessageStore {
	return &AIChatMessageStore{db: db}
}

func (s *AIChatMessageStore) Create(ctx context.Context, message *domain.AIChatMessage) error {
	if message == nil {
		return errors.New("message required")
	}
	if message.ID == "" {
		return errors.New("message id required")
	}
	return s.db.WithContext(ctx).Create(message).Error
}

func (s *AIChatMessageStore) ListBySession(ctx context.Context, sessionID string, limit int, before time.Time) ([]domain.AIChatMessage, error) {
	if sessionID == "" {
		return nil, errors.New("session_id required")
	}

	query := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID)

	if !before.IsZero() {
		query = query.Where("created_at < ?", before)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	query = query.Order("created_at DESC")

	var messages []domain.AIChatMessage
	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

var _ repository.AIChatMessageRepository = (*AIChatMessageStore)(nil)

func (s *AIChatMessageStore) CountUserMessagesSince(ctx context.Context, accountID string, since time.Time) (int64, error) {
	if accountID == "" {
		return 0, errors.New("account_id required")
	}

	query := s.db.WithContext(ctx).
		Model(&domain.AIChatMessage{}).
		Joins("JOIN ai_chat_sessions ON ai_chat_sessions.id = ai_chat_messages.session_id").
		Where("ai_chat_sessions.account_id = ?", accountID).
		Where("ai_chat_messages.sender = ?", "user")

	if !since.IsZero() {
		query = query.Where("ai_chat_messages.created_at >= ?", since)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *AIChatMessageStore) UsageStatsByAccountSince(ctx context.Context, accountID string, since time.Time) (repository.AIChatUsageStats, error) {
	if accountID == "" {
		return repository.AIChatUsageStats{}, errors.New("account_id required")
	}

	query := s.db.WithContext(ctx).
		Model(&domain.AIChatMessage{}).
		Select(`
		SUM(CASE WHEN ai_chat_messages.sender = 'user' THEN 1 ELSE 0 END) AS user_messages,
		SUM(CASE WHEN ai_chat_messages.sender = 'assistant' THEN 1 ELSE 0 END) AS assistant_messages,
		SUM(ai_chat_messages.prompt_tokens) AS prompt_tokens,
		SUM(ai_chat_messages.result_tokens) AS result_tokens`).
		Joins("JOIN ai_chat_sessions ON ai_chat_sessions.id = ai_chat_messages.session_id").
		Where("ai_chat_sessions.account_id = ?", accountID)

	if !since.IsZero() {
		query = query.Where("ai_chat_messages.created_at >= ?", since)
	}

	type row struct {
		UserMessages      sql.NullInt64
		AssistantMessages sql.NullInt64
		PromptTokens      sql.NullInt64
		ResultTokens      sql.NullInt64
	}

	var r row
	if err := query.Scan(&r).Error; err != nil {
		return repository.AIChatUsageStats{}, err
	}

	convert := func(n sql.NullInt64) int64 {
		if !n.Valid {
			return 0
		}
		return n.Int64
	}

	return repository.AIChatUsageStats{
		UserMessages:      convert(r.UserMessages),
		AssistantMessages: convert(r.AssistantMessages),
		PromptTokens:      convert(r.PromptTokens),
		ResultTokens:      convert(r.ResultTokens),
	}, nil
}

func (s *AIChatMessageStore) UsageStatsBySchoolSince(ctx context.Context, schoolID string, since time.Time, role domain.Role, limit int, offset int, sort repository.AIChatUsageSort) ([]repository.AIChatAccountUsage, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	query := s.db.WithContext(ctx).
		Model(&domain.AIChatMessage{}).
		Select(`
		ai_chat_sessions.account_id AS account_id,
		SUM(CASE WHEN ai_chat_messages.sender = 'user' THEN 1 ELSE 0 END) AS user_messages,
		SUM(CASE WHEN ai_chat_messages.sender = 'assistant' THEN 1 ELSE 0 END) AS assistant_messages,
		SUM(ai_chat_messages.prompt_tokens) AS prompt_tokens,
		SUM(ai_chat_messages.result_tokens) AS result_tokens`).
		Joins("JOIN ai_chat_sessions ON ai_chat_sessions.id = ai_chat_messages.session_id").
		Where("ai_chat_sessions.school_id = ?", schoolID).
		Group("ai_chat_sessions.account_id")

	if !since.IsZero() {
		query = query.Where("ai_chat_messages.created_at >= ?", since)
	}

	if role != "" {
		query = query.Where("ai_chat_sessions.role = ?", role)
	}

	orderExpr := "user_messages"
	switch sort.Field {
	case repository.AIChatUsageSortTotalMessages:
		orderExpr = "(user_messages + assistant_messages)"
	case repository.AIChatUsageSortTotalTokens:
		orderExpr = "(prompt_tokens + result_tokens)"
	default:
		orderExpr = "user_messages"
	}
	orderDir := "DESC"
	if sort.Direction == repository.SortDirectionAsc {
		orderDir = "ASC"
	}
	query = query.Order(orderExpr + " " + orderDir)

	if offset > 0 {
		query = query.Offset(offset)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	type row struct {
		AccountID         string
		UserMessages      sql.NullInt64
		AssistantMessages sql.NullInt64
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

	result := make([]repository.AIChatAccountUsage, 0, len(rows))
	for _, r := range rows {
		result = append(result, repository.AIChatAccountUsage{
			AccountID: r.AccountID,
			AIChatUsageStats: repository.AIChatUsageStats{
				UserMessages:      convert(r.UserMessages),
				AssistantMessages: convert(r.AssistantMessages),
				PromptTokens:      convert(r.PromptTokens),
				ResultTokens:      convert(r.ResultTokens),
			},
		})
	}

	return result, nil
}

func (s *AIChatMessageStore) UsageTotalsBySchoolSince(ctx context.Context, schoolID string, since time.Time, role domain.Role) (repository.AIChatUsageTotals, error) {
	if schoolID == "" {
		return repository.AIChatUsageTotals{}, errors.New("school_id required")
	}

	query := s.db.WithContext(ctx).
		Model(&domain.AIChatMessage{}).
		Select(`
		COUNT(DISTINCT ai_chat_sessions.account_id) AS account_count,
		SUM(CASE WHEN ai_chat_messages.sender = 'user' THEN 1 ELSE 0 END) AS user_messages,
		SUM(CASE WHEN ai_chat_messages.sender = 'assistant' THEN 1 ELSE 0 END) AS assistant_messages,
		SUM(ai_chat_messages.prompt_tokens) AS prompt_tokens,
		SUM(ai_chat_messages.result_tokens) AS result_tokens`).
		Joins("JOIN ai_chat_sessions ON ai_chat_sessions.id = ai_chat_messages.session_id").
		Where("ai_chat_sessions.school_id = ?", schoolID)

	if !since.IsZero() {
		query = query.Where("ai_chat_messages.created_at >= ?", since)
	}

	if role != "" {
		query = query.Where("ai_chat_sessions.role = ?", role)
	}

	type row struct {
		AccountCount      sql.NullInt64
		UserMessages      sql.NullInt64
		AssistantMessages sql.NullInt64
		PromptTokens      sql.NullInt64
		ResultTokens      sql.NullInt64
	}

	var r row
	if err := query.Scan(&r).Error; err != nil {
		return repository.AIChatUsageTotals{}, err
	}

	convert := func(n sql.NullInt64) int64 {
		if !n.Valid {
			return 0
		}
		return n.Int64
	}

	return repository.AIChatUsageTotals{
		AccountCount: convert(r.AccountCount),
		AIChatUsageStats: repository.AIChatUsageStats{
			UserMessages:      convert(r.UserMessages),
			AssistantMessages: convert(r.AssistantMessages),
			PromptTokens:      convert(r.PromptTokens),
			ResultTokens:      convert(r.ResultTokens),
		},
	}, nil
}

func (s *AIChatMessageStore) UsageByRoleSince(ctx context.Context, schoolID string, since time.Time) (map[domain.Role]repository.AIChatUsageTotals, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	query := s.db.WithContext(ctx).
		Model(&domain.AIChatMessage{}).
		Select(`
		ai_chat_sessions.role AS role,
		COUNT(DISTINCT ai_chat_sessions.account_id) AS account_count,
		SUM(CASE WHEN ai_chat_messages.sender = 'user' THEN 1 ELSE 0 END) AS user_messages,
		SUM(CASE WHEN ai_chat_messages.sender = 'assistant' THEN 1 ELSE 0 END) AS assistant_messages,
		SUM(ai_chat_messages.prompt_tokens) AS prompt_tokens,
		SUM(ai_chat_messages.result_tokens) AS result_tokens`).
		Joins("JOIN ai_chat_sessions ON ai_chat_sessions.id = ai_chat_messages.session_id").
		Where("ai_chat_sessions.school_id = ?", schoolID).
		Group("ai_chat_sessions.role")

	if !since.IsZero() {
		query = query.Where("ai_chat_messages.created_at >= ?", since)
	}

	type row struct {
		Role              string
		AccountCount      sql.NullInt64
		UserMessages      sql.NullInt64
		AssistantMessages sql.NullInt64
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
		result[role] = repository.AIChatUsageTotals{
			AccountCount: convert(r.AccountCount),
			AIChatUsageStats: repository.AIChatUsageStats{
				UserMessages:      convert(r.UserMessages),
				AssistantMessages: convert(r.AssistantMessages),
				PromptTokens:      convert(r.PromptTokens),
				ResultTokens:      convert(r.ResultTokens),
			},
		}
	}

	return result, nil
}

func (s *AIChatMessageStore) UsageTimelineBySchool(ctx context.Context, schoolID string, start time.Time, end time.Time, role domain.Role) ([]repository.AIChatUsageTimelinePoint, error) {
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	query := s.db.WithContext(ctx).
		Model(&domain.AIChatMessage{}).
		Select(`
		DATE_TRUNC('day', ai_chat_messages.created_at) AS bucket,
		COUNT(DISTINCT ai_chat_sessions.account_id) AS account_count,
		SUM(CASE WHEN ai_chat_messages.sender = 'user' THEN 1 ELSE 0 END) AS user_messages,
		SUM(CASE WHEN ai_chat_messages.sender = 'assistant' THEN 1 ELSE 0 END) AS assistant_messages,
		SUM(ai_chat_messages.prompt_tokens) AS prompt_tokens,
		SUM(ai_chat_messages.result_tokens) AS result_tokens`).
		Joins("JOIN ai_chat_sessions ON ai_chat_sessions.id = ai_chat_messages.session_id").
		Where("ai_chat_sessions.school_id = ?", schoolID).
		Group("bucket").
		Order("bucket ASC")

	if !start.IsZero() {
		query = query.Where("ai_chat_messages.created_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("ai_chat_messages.created_at < ?", end)
	}
	if role != "" {
		query = query.Where("ai_chat_sessions.role = ?", role)
	}

	type row struct {
		Bucket            time.Time
		AccountCount      sql.NullInt64
		UserMessages      sql.NullInt64
		AssistantMessages sql.NullInt64
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
		result = append(result, repository.AIChatUsageTimelinePoint{
			Bucket:       r.Bucket,
			AccountCount: convert(r.AccountCount),
			AIChatUsageStats: repository.AIChatUsageStats{
				UserMessages:      convert(r.UserMessages),
				AssistantMessages: convert(r.AssistantMessages),
				PromptTokens:      convert(r.PromptTokens),
				ResultTokens:      convert(r.ResultTokens),
			},
		})
	}

	return result, nil
}
