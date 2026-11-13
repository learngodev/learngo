package usecase

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"learn-go/internal/domain/aiagent"
)

// AIChatModel 抽象各提供商的对话生成能力。
type AIChatModel interface {
	GenerateResponse(ctx context.Context, req AIChatModelRequest) (*AIChatModelResponse, error)
	StreamResponse(ctx context.Context, req AIChatModelRequest, onChunk func(chunk string) error) (*AIChatModelResponse, error)
}

// AIChatModelRequest 携带生成助手回复所需的上下文。
type AIChatModelRequest struct {
	Setting *aiagent.AIAgentSetting
	Message string
}

// AIChatModelResponse 包含助手输出及用量信息。
type AIChatModelResponse struct {
	Content      string
	PromptTokens int
	ResultTokens int
	Latency      time.Duration
	Reason       ProviderErrorReason
}

// ProviderErrorReason 表示提供商调用失败的原因。
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

// ProviderError 对外部提供商的失败进行分类。
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

// NormalizeProviderError 将任意错误转换为 ProviderError。
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

// OpenAIChatModel 基于 OpenAI 兼容接口实现 AIChatModel。
type OpenAIChatModel struct {
	client *http.Client
}

// NewOpenAIChatModel 构造 OpenAI 兼容模型。
func NewOpenAIChatModel() AIChatModel {
	return &OpenAIChatModel{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateResponse 调用已配置的提供商接口。
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
		case aiagent.AIProviderQwen:
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		case aiagent.AIProviderDeepSeek:
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
		Reason:       ProviderErrorReasonUnknown, // 成功
	}, nil
}

// StreamResponse 以流式方式调用已配置的提供商接口。
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
		case aiagent.AIProviderQwen:
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		case aiagent.AIProviderDeepSeek:
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

	// 添加历史消息

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
	// 注意：并非所有提供商都会在流式响应的最后一块中返回用量统计。
	// 可能需要估算，或直接返回 0。

	return &AIChatModelResponse{
		Content:      content,
		PromptTokens: 0, // 流式模式下未知
		ResultTokens: 0, // 流式模式下未知
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

// EchoAIChatModel 是开发用的占位模型，会回显用户意图。
type EchoAIChatModel struct{}

// NewEchoAIChatModel 构造默认的桩模型。
func NewEchoAIChatModel() AIChatModel {
	return &EchoAIChatModel{}
}

// GenerateResponse 通过返回固定回复实现 AIChatModel。
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

// StreamResponse 通过模拟流式输出实现 AIChatModel。
func (m *EchoAIChatModel) StreamResponse(ctx context.Context, req AIChatModelRequest, onChunk func(chunk string) error) (*AIChatModelResponse, error) {
	start := time.Now()

	trimmed := strings.TrimSpace(req.Message)
	if trimmed == "" {
		trimmed = "(空消息)"
	}

	content := fmt.Sprintf("这是一个示例回答，用于说明 AI 接口尚未接入。你刚才的问题是：%s", trimmed)

	// 模拟流式输出
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
	// 粗略估算：按每 4 个字符 1 个 token 计算。
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
