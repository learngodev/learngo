package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	aibiz "learn-go/internal/biz/ai"
	"net/http"
	"strings"
	"time"
)

// AIChatModel abstracts provider-specific chat generation.
type AIChatModel interface {
	GenerateResponse(ctx context.Context, req AIChatModelRequest) (*AIChatModelResponse, error)
	StreamResponse(ctx context.Context, req AIChatModelRequest, onChunk func(chunk string) error) (*AIChatModelResponse, error)
}

// AIChatModelRequest carries the context for generating an assistant reply.
type AIChatModelRequest struct {
	Setting *aibiz.AIAgentSetting
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

// OpenAIChatModel implements AIChatModel using OpenAI-compatible API.
type OpenAIChatModel struct {
	client *http.Client
}

// NewOpenAIChatModel constructs the OpenAI-compatible model.
func NewOpenAIChatModel() AIChatModel {
	return &OpenAIChatModel{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateResponse calls the configured provider API.
func (m *OpenAIChatModel) GenerateResponse(ctx context.Context, req AIChatModelRequest) (*AIChatModelResponse, error) {
	start := time.Now()

	if req.Setting == nil {
		return nil, &ProviderError{Reason: ProviderErrorReasonInput, Message: "ai setting missing"}
	}
	if req.Setting.APIKey == "" {
		return nil, &ProviderError{Reason: ProviderErrorReasonAuth, Message: "api key missing"}
	}

	baseURL := strings.TrimRight(req.Setting.BaseURL, "/")
	if baseURL == "" {
		switch req.Setting.Provider {
		case aibiz.AIProviderQwen:
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		case aibiz.AIProviderDeepSeek:
			baseURL = "https://api.deepseek.com"
		default:
			return nil, &ProviderError{Reason: ProviderErrorReasonInput, Message: "base url required for custom provider"}
		}
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	messages := []openAIMessage{}
	if req.Setting.SystemPrompt != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: req.Setting.SystemPrompt})
	}

	messages = append(messages, openAIMessage{Role: "user", Content: req.Message})

	payload := openAIChatRequest{
		Model:       req.Setting.Model,
		Messages:    messages,
		Temperature: req.Setting.Temperature,
		TopP:        req.Setting.TopP,
		MaxTokens:   req.Setting.MaxOutputTokens,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, &ProviderError{Reason: ProviderErrorReasonInput, Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, &ProviderError{Reason: ProviderErrorReasonTransport, Err: err}
	}

	httpReq.Header.Set("Content-Type", "application/json")

	apiKey := strings.TrimSpace(req.Setting.APIKey)
	if strings.HasPrefix(strings.ToLower(apiKey), "bearer ") {
		apiKey = strings.TrimSpace(apiKey[7:])
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := m.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, &ProviderError{Reason: ProviderErrorReasonTimeout, Err: err}
		}
		return nil, &ProviderError{Reason: ProviderErrorReasonTransport, Err: err}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp openAIChatResponse
		_ = json.Unmarshal(respBody, &errResp)
		msg := fmt.Sprintf("provider error: %d", resp.StatusCode)
		if errResp.Error != nil {
			msg = fmt.Sprintf("%s: %s", msg, errResp.Error.Message)
		}
		reason := ProviderErrorReasonUpstream
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			reason = ProviderErrorReasonAuth
		} else if resp.StatusCode == http.StatusTooManyRequests {
			reason = ProviderErrorReasonQuota
		} else if resp.StatusCode >= 500 {
			reason = ProviderErrorReasonUpstream
		}
		return nil, &ProviderError{Reason: reason, Message: msg}
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, &ProviderError{Reason: ProviderErrorReasonUpstream, Message: "invalid response format", Err: err}
	}

	if len(chatResp.Choices) == 0 {
		return nil, &ProviderError{Reason: ProviderErrorReasonUpstream, Message: "no choices returned"}
	}

	content := chatResp.Choices[0].Message.Content
	return &AIChatModelResponse{
		Content:      content,
		PromptTokens: chatResp.Usage.PromptTokens,
		ResultTokens: chatResp.Usage.CompletionTokens,
		Latency:      time.Since(start),
		Reason:       ProviderErrorReasonUnknown, // Success
	}, nil
}

// StreamResponse calls the configured provider API with streaming enabled.
func (m *OpenAIChatModel) StreamResponse(ctx context.Context, req AIChatModelRequest, onChunk func(chunk string) error) (*AIChatModelResponse, error) {
	start := time.Now()

	if req.Setting == nil {
		return nil, &ProviderError{Reason: ProviderErrorReasonInput, Message: "ai setting missing"}
	}
	if req.Setting.APIKey == "" {
		return nil, &ProviderError{Reason: ProviderErrorReasonAuth, Message: "api key missing"}
	}

	baseURL := strings.TrimRight(req.Setting.BaseURL, "/")
	if baseURL == "" {
		switch req.Setting.Provider {
		case aibiz.AIProviderQwen:
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		case aibiz.AIProviderDeepSeek:
			baseURL = "https://api.deepseek.com"
		default:
			return nil, &ProviderError{Reason: ProviderErrorReasonInput, Message: "base url required for custom provider"}
		}
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	messages := []openAIMessage{}
	if req.Setting.SystemPrompt != "" {
		messages = append(messages, openAIMessage{Role: "system", Content: req.Setting.SystemPrompt})
	}

	// Add history messages

	messages = append(messages, openAIMessage{Role: "user", Content: req.Message})

	payload := openAIChatRequest{
		Model:       req.Setting.Model,
		Messages:    messages,
		Temperature: req.Setting.Temperature,
		TopP:        req.Setting.TopP,
		MaxTokens:   req.Setting.MaxOutputTokens,
		Stream:      true,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, &ProviderError{Reason: ProviderErrorReasonInput, Err: err}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, &ProviderError{Reason: ProviderErrorReasonTransport, Err: err}
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	apiKey := strings.TrimSpace(req.Setting.APIKey)
	if strings.HasPrefix(strings.ToLower(apiKey), "bearer ") {
		apiKey = strings.TrimSpace(apiKey[7:])
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := m.client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, &ProviderError{Reason: ProviderErrorReasonTimeout, Err: err}
		}
		return nil, &ProviderError{Reason: ProviderErrorReasonTransport, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		var errResp openAIChatResponse
		_ = json.Unmarshal(respBody, &errResp)
		msg := fmt.Sprintf("provider error: %d", resp.StatusCode)
		if errResp.Error != nil {
			msg = fmt.Sprintf("%s: %s", msg, errResp.Error.Message)
		}
		reason := ProviderErrorReasonUpstream
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			reason = ProviderErrorReasonAuth
		} else if resp.StatusCode == http.StatusTooManyRequests {
			reason = ProviderErrorReasonQuota
		} else if resp.StatusCode >= 500 {
			reason = ProviderErrorReasonUpstream
		}
		return nil, &ProviderError{Reason: reason, Message: msg}
	}

	reader := bufio.NewReader(resp.Body)
	var fullContent strings.Builder

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, &ProviderError{Reason: ProviderErrorReasonTransport, Err: err}
		}

		lineStr := strings.TrimSpace(string(line))
		if lineStr == "" {
			continue
		}
		if !strings.HasPrefix(lineStr, "data: ") {
			continue
		}
		data := strings.TrimPrefix(lineStr, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openAIChatStreamResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta.Content
			if delta != "" {
				fullContent.WriteString(delta)
				if err := onChunk(delta); err != nil {
					return nil, err
				}
			}
		}
	}

	content := fullContent.String()
	// Note: Stream response usually doesn't include usage stats in the last chunk for all providers.
	// We might need to estimate or just return 0.

	return &AIChatModelResponse{
		Content:      content,
		PromptTokens: 0, // Unknown in stream
		ResultTokens: 0, // Unknown in stream
		Latency:      time.Since(start),
		Reason:       ProviderErrorReasonUnknown,
	}, nil
}

type openAIChatStreamResponse struct {
	Choices []openAIChatStreamChoice `json:"choices"`
}

type openAIChatStreamChoice struct {
	Delta openAIChatStreamDelta `json:"delta"`
}

type openAIChatStreamDelta struct {
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float32         `json:"temperature,omitempty"`
	TopP        float32         `json:"top_p,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Choices []openAIChoice     `json:"choices"`
	Usage   openAIUsage        `json:"usage"`
	Error   *openAIErrorDetail `json:"error,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIErrorDetail struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Code    interface{} `json:"code"`
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

// StreamResponse implements AIChatModel by simulating streaming.
func (m *EchoAIChatModel) StreamResponse(ctx context.Context, req AIChatModelRequest, onChunk func(chunk string) error) (*AIChatModelResponse, error) {
	start := time.Now()

	trimmed := strings.TrimSpace(req.Message)
	if trimmed == "" {
		trimmed = "(空消息)"
	}

	content := fmt.Sprintf("这是一个示例回答，用于说明 AI 接口尚未接入。你刚才的问题是：%s", trimmed)

	// Simulate streaming
	runes := []rune(content)
	chunkSize := 5
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		if err := onChunk(chunk); err != nil {
			return nil, err
		}
		time.Sleep(50 * time.Millisecond)
	}

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
