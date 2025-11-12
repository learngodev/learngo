package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"learn-go/internal/domain"
)

// AIChatModel abstracts provider-specific chat generation.
type AIChatModel interface {
	GenerateResponse(ctx context.Context, req AIChatModelRequest) (*AIChatModelResponse, error)
}

// AIChatModelRequest carries the context for generating an assistant reply.
type AIChatModelRequest struct {
	Setting *domain.AIAgentSetting
	Session *domain.AIChatSession
	Message string
}

// AIChatModelResponse contains the assistant output and usage metadata.
type AIChatModelResponse struct {
	Content      string
	PromptTokens int
	ResultTokens int
	Latency      time.Duration
	Reason       ProviderErrorReason
}

// ProviderErrorReason indicates why a provider call failed.
type ProviderErrorReason string

const (
	ProviderErrorReasonUnknown   ProviderErrorReason = "unknown"
	ProviderErrorReasonUpstream  ProviderErrorReason = "upstream"
	ProviderErrorReasonQuota     ProviderErrorReason = "quota"
	ProviderErrorReasonInput     ProviderErrorReason = "input"
	ProviderErrorReasonAuth      ProviderErrorReason = "auth"
	ProviderErrorReasonTimeout   ProviderErrorReason = "timeout"
	ProviderErrorReasonTransport ProviderErrorReason = "transport"
)

// ProviderError categorizes external provider failures.
type ProviderError struct {
	Reason  ProviderErrorReason
	Message string
	Err     error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Reason)
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// NormalizeProviderError converts an arbitrary error into a ProviderError.
func NormalizeProviderError(err error) *ProviderError {
	if err == nil {
		return nil
	}
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe
	}
	return &ProviderError{Reason: ProviderErrorReasonUnknown, Err: err}
}

// EchoAIChatModel is a placeholder model that echoes user intent for development purposes.
type EchoAIChatModel struct{}

// NewEchoAIChatModel constructs the default stub model.
func NewEchoAIChatModel() AIChatModel {
	return &EchoAIChatModel{}
}

// GenerateResponse implements AIChatModel by returning a canned response.
func (m *EchoAIChatModel) GenerateResponse(ctx context.Context, req AIChatModelRequest) (*AIChatModelResponse, error) {
	start := time.Now()

	trimmed := strings.TrimSpace(req.Message)
	if trimmed == "" {
		trimmed = "(空消息)"
	}

	content := fmt.Sprintf("这是一个示例回答，用于说明 AI 接口尚未接入。你刚才的问题是：%s", trimmed)

	promptTokens := estimateTokens(req.Message)
	resultTokens := estimateTokens(content)

	return &AIChatModelResponse{
		Content:      content,
		PromptTokens: promptTokens,
		ResultTokens: resultTokens,
		Latency:      time.Since(start),
		Reason:       ProviderErrorReasonUnknown,
	}, nil
}

func estimateTokens(text string) int {
	// Rough heuristic: assume 4 characters per token.
	length := len([]rune(strings.TrimSpace(text)))
	if length == 0 {
		return 1
	}
	tokens := length / 4
	if tokens == 0 {
		tokens = 1
	}
	return tokens
}
