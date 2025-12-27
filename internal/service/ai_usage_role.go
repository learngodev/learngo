package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

var usageRoleOrder = []domain.Role{domain.RoleTeacher, domain.RoleStudent, domain.RoleAdmin}

// AIChatUsageRoleSummary exposes usage totals for a specific role.
type AIChatUsageRoleSummary struct {
	Role              domain.Role
	AccountCount      int
	UserMessages      int64
	AssistantMessages int64
	TotalMessages     int64
	PromptTokens      int64
	ResultTokens      int64
	TotalTokens       int64
	MessageShare      float64
	TokenShare        float64
	AverageMessages   float64
	AverageTokens     float64
}

// GetUsageRoleBreakdown returns usage totals split by role for a school.
func (s *AIAssistantService) GetUsageRoleBreakdown(ctx context.Context, input ListUsageSummariesInput) ([]AIChatUsageRoleSummary, error) {
	schoolID := strings.TrimSpace(input.SchoolID)
	if schoolID == "" {
		return nil, errors.New("school_id required")
	}

	if _, err := s.settings.GetBySchoolID(ctx, schoolID); err != nil {
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

	breakdown, err := s.messages.UsageByRoleSince(ctx, schoolID, since)
	if err != nil {
		return nil, err
	}

	logBreakdown, err := s.logs.UsageByRoleSince(ctx, schoolID, since)
	if err != nil {
		return nil, err
	}

	if breakdown == nil {
		breakdown = make(map[domain.Role]repository.AIChatUsageTotals)
	}

	for role, totals := range logBreakdown {
		existing, exists := breakdown[role]
		if !exists {
			breakdown[role] = totals
		} else {
			if totals.AccountCount > existing.AccountCount {
				existing.AccountCount = totals.AccountCount
			}
			existing.UserMessages += totals.UserMessages
			existing.AssistantMessages += totals.AssistantMessages
			existing.PromptTokens += totals.PromptTokens
			existing.ResultTokens += totals.ResultTokens
			breakdown[role] = existing
		}
	}

	for _, role := range usageRoleOrder {
		if _, exists := breakdown[role]; !exists {
			breakdown[role] = repository.AIChatUsageTotals{}
		}
	}

	totalMessagesAll := int64(0)
	totalTokensAll := int64(0)
	for _, totals := range breakdown {
		totalMessagesAll += totals.UserMessages + totals.AssistantMessages
		totalTokensAll += totals.PromptTokens + totals.ResultTokens
	}

	orderedRoles := make([]domain.Role, 0, len(breakdown))
	seen := make(map[domain.Role]bool, len(breakdown))
	for _, role := range usageRoleOrder {
		if _, exists := breakdown[role]; exists {
			orderedRoles = append(orderedRoles, role)
			seen[role] = true
		}
	}

	extra := make([]domain.Role, 0)
	for role := range breakdown {
		if !seen[role] {
			extra = append(extra, role)
		}
	}

	sort.Slice(extra, func(i, j int) bool {
		return string(extra[i]) < string(extra[j])
	})
	orderedRoles = append(orderedRoles, extra...)

	summaries := make([]AIChatUsageRoleSummary, 0, len(orderedRoles))
	for _, role := range orderedRoles {
		totals := breakdown[role]
		totalMessages := totals.UserMessages + totals.AssistantMessages
		totalTokens := totals.PromptTokens + totals.ResultTokens
		messageShare := 0.0
		if totalMessagesAll > 0 {
			messageShare = float64(totalMessages) / float64(totalMessagesAll)
		}
		tokenShare := 0.0
		if totalTokensAll > 0 {
			tokenShare = float64(totalTokens) / float64(totalTokensAll)
		}
		avgMessages := 0.0
		avgTokens := 0.0
		if totals.AccountCount > 0 {
			avgMessages = float64(totalMessages) / float64(totals.AccountCount)
			avgTokens = float64(totalTokens) / float64(totals.AccountCount)
		}

		summaries = append(summaries, AIChatUsageRoleSummary{
			Role:              role,
			AccountCount:      int(totals.AccountCount),
			UserMessages:      totals.UserMessages,
			AssistantMessages: totals.AssistantMessages,
			TotalMessages:     totalMessages,
			PromptTokens:      totals.PromptTokens,
			ResultTokens:      totals.ResultTokens,
			TotalTokens:       totalTokens,
			MessageShare:      messageShare,
			TokenShare:        tokenShare,
			AverageMessages:   avgMessages,
			AverageTokens:     avgTokens,
		})
	}

	return summaries, nil
}
