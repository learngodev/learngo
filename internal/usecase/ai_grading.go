package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/internal/domain/aiagent"
	"learn-go/internal/domain/identity"
)

// AIGradingUsecase 提供基于 AI 的作业检查与批改。
type AIGradingUsecase struct {
	settings aiagent.AIAgentSettingRepository
	logs     aiagent.AIUsageLogRepository
	model    AIChatModel
}

// NewAIGradingUsecase 创建 AIGradingUsecase。
func NewAIGradingUsecase(
	settings aiagent.AIAgentSettingRepository,
	logs aiagent.AIUsageLogRepository,
	model AIChatModel,
) *AIGradingUsecase {
	return &AIGradingUsecase{
		settings: settings,
		logs:     logs,
		model:    model,
	}
}

// CheckAssignmentInput 包含 AI 提交前检查所需的数据。
type CheckAssignmentInput struct {
	SchoolID    string
	AccountID   string
	Role        identity.Role
	Title       string
	Description string
	Content     string
}

// CheckAssignmentResult 包含给学生的 AI 反馈。
type CheckAssignmentResult struct {
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
	Overall     string   `json:"overall"`
}

// ExplainQuestionInput 包含 AI 讲题所需的数据。
// 用于像老师一样讲解题目，但不给出最终答案。
type ExplainQuestionInput struct {
	SchoolID     string
	AccountID    string
	Role         identity.Role
	Title        string
	Prompt       string
	QuestionType string
	Options      []string
	ExtraContext string
}

// ExplainQuestionResult 包含 AI 对题目的讲解。
// 注意：不得包含最终答案。
type ExplainQuestionResult struct {
	Analysis  string   `json:"analysis"`
	Steps     []string `json:"steps"`
	KeyPoints []string `json:"key_points"`
	Pitfalls  []string `json:"pitfalls"`
	Checklist []string `json:"checklist"`
}

// GradeAssignmentInput 包含 AI 批改所需的数据。
type GradeAssignmentInput struct {
	AccountID   string
	Role        identity.Role
	SchoolID    string
	Title       string
	Description string
	Content     string
	Rubrics     string
}

// GradeAssignmentResult 包含给教师的 AI 批改结果。
type GradeAssignmentResult struct {
	Score       int      `json:"score"`
	Summary     string   `json:"summary"`
	Suggestions []string `json:"suggestions"`
	ItemScores  []int    `json:"item_scores"`
}

// CheckAssignment 为学生执行作业提交前检查。
func (s *AIGradingUsecase) CheckAssignment(ctx context.Context, input CheckAssignmentInput) (*CheckAssignmentResult, error) {
	setting, err := s.getAISetting(ctx, input.SchoolID)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`
You are a helpful teaching assistant. Please review the following assignment submission (or a single question answer) and provide feedback to the student in Chinese (Simplified).
Focus on:
1. Relevance to the topic.
2. Clarity and structure.
3. Grammar and spelling (if applicable).
4. Word count requirements (if applicable).

Assignment/Question Title: %s
Description/Prompt: %s

Student Submission:
%s

Please return the result in strict JSON format as follows (ensure all text fields are in Chinese):
{
  "issues": ["issue 1", "issue 2"],
  "suggestions": ["suggestion 1", "suggestion 2"],
  "overall": "overall comment"
}
`, input.Title, input.Description, input.Content)

	req := AIChatModelRequest{
		Setting: setting,
		Message: prompt,
	}

	resp, err := s.model.GenerateResponse(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.logs.Create(ctx, &aiagent.AIUsageLog{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		AccountID:    input.AccountID,
		Role:         input.Role,
		Feature:      "assignment_check",
		Model:        setting.Model,
		PromptTokens: resp.PromptTokens,
		ResultTokens: resp.ResultTokens,
		TotalTokens:  resp.PromptTokens + resp.ResultTokens,
		CreatedAt:    time.Now(),
	})

	var result CheckAssignmentResult
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &result, nil
}

// ExplainQuestion 像老师一样（用简体中文）讲解题目，但不给出最终答案。
func (s *AIGradingUsecase) ExplainQuestion(ctx context.Context, input ExplainQuestionInput) (*ExplainQuestionResult, error) {
	setting, err := s.getAISetting(ctx, input.SchoolID)
	if err != nil {
		return nil, err
	}

	optionsText := "(none)"
	if len(input.Options) > 0 {
		// 选项按普通行列出，不暗示任何正确选项。
		optionsText = strings.Join(input.Options, "\n")
	}

	extra := strings.TrimSpace(input.ExtraContext)
	if extra == "" {
		extra = "(none)"
	}

	prompt := fmt.Sprintf(`
You are an experienced teacher. Please explain the following question to a student in Chinese (Simplified).

IMPORTANT RULES (must follow strictly):
1) DO NOT provide the final answer, the correct option, or the final numeric result.
2) DO NOT write a complete, copy-pastable final solution/code. If needed, only provide high-level pseudocode or thinking steps.
3) Focus on understanding the question, key concepts, reasoning steps, and common pitfalls.
4) If the question is multiple-choice, explain how to analyze and eliminate options WITHOUT revealing which one is correct.
5) Keep the tone supportive and teacher-like.

Assignment Title: %s
Question Type: %s

Question Prompt:
%s

Options (if any):
%s

Extra Context (if any):
%s

Return the result in STRICT JSON format, all fields in Chinese (Simplified):
{
  "analysis": "题意解析（用自己的话解释题目在问什么）",
  "steps": ["思路步骤 1", "思路步骤 2"],
  "key_points": ["关键知识点 1", "关键知识点 2"],
  "pitfalls": ["易错点 1", "易错点 2"],
  "checklist": ["自查项 1", "自查项 2"]
}
`, input.Title, input.QuestionType, input.Prompt, optionsText, extra)

	req := AIChatModelRequest{
		Setting: setting,
		Message: prompt,
	}

	resp, err := s.model.GenerateResponse(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.logs.Create(ctx, &aiagent.AIUsageLog{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		AccountID:    input.AccountID,
		Role:         input.Role,
		Feature:      "question_explain",
		Model:        setting.Model,
		PromptTokens: resp.PromptTokens,
		ResultTokens: resp.ResultTokens,
		TotalTokens:  resp.PromptTokens + resp.ResultTokens,
		CreatedAt:    time.Now(),
	})

	var result ExplainQuestionResult
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &result, nil
}

// GradeAssignment 为教师执行 AI 批改。
func (s *AIGradingUsecase) GradeAssignment(ctx context.Context, input GradeAssignmentInput) (*GradeAssignmentResult, error) {
	setting, err := s.getAISetting(ctx, input.SchoolID)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`
You are a professional teacher. Please grade the following assignment submission based on the requirements.
Please provide all feedback, summary, and suggestions in Chinese (Simplified).

Assignment Title: %s
Assignment Description: %s
Rubrics/Requirements: %s

Student Submission:
%s

Please return the result in strict JSON format as follows (ensure all text fields are in Chinese):
{
  "score": 85, // Integer 0-100
  "summary": "Brief summary of the grading",
  "suggestions": ["suggestion 1", "suggestion 2"],
  "item_scores": [10, 15, 20] // Scores for each question in the order they appear
}
`, input.Title, input.Description, input.Rubrics, input.Content)

	req := AIChatModelRequest{
		Setting: setting,
		Message: prompt,
	}

	resp, err := s.model.GenerateResponse(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.logs.Create(ctx, &aiagent.AIUsageLog{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		AccountID:    input.AccountID,
		Role:         input.Role,
		Feature:      "assignment_grading",
		Model:        setting.Model,
		PromptTokens: resp.PromptTokens,
		ResultTokens: resp.ResultTokens,
		TotalTokens:  resp.PromptTokens + resp.ResultTokens,
		CreatedAt:    time.Now(),
	})

	var result GradeAssignmentResult
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &result, nil
}

// GenerateQuestionsInput 包含 AI 出题的参数。
type GenerateQuestionsInput struct {
	AccountID  string
	Role       identity.Role
	SchoolID   string
	Topic      string
	Count      int
	Difficulty string
}

// GeneratedQuestion 表示 AI 生成的一道题目。
type GeneratedQuestion struct {
	Type    string   `json:"type"` // choice（选择题）、essay（问答题）
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"` // 仅选择题使用
	Answer  string   `json:"answer"`
	Score   float64  `json:"score"`
}

// GenerateQuestionsResult 包含生成的题目列表。
type GenerateQuestionsResult struct {
	Questions []GeneratedQuestion `json:"questions"`
}

// GenerateQuestions 根据主题生成作业题目。
func (s *AIGradingUsecase) GenerateQuestions(ctx context.Context, input GenerateQuestionsInput) (*GenerateQuestionsResult, error) {
	setting, err := s.getAISetting(ctx, input.SchoolID)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`
You are a helpful teacher. Please generate %d assignment questions about "%s".
Difficulty level: %s.

Requirements:
1. Mix of single choice questions and essay questions.
2. For choice questions, provide 4 options.
3. Provide the correct answer for each question.
4. Assign a reasonable score for each question (total score doesn't need to be 100).

Please return the result in strict JSON format as follows:
{
  "questions": [
    {
      "type": "choice", // or "essay"
      "prompt": "Question text here",
      "options": ["Option A", "Option B", "Option C", "Option D"], // Empty for essay
      "answer": "Correct answer text",
      "score": 10
    }
  ]
}
`, input.Count, input.Topic, input.Difficulty)

	req := AIChatModelRequest{
		Setting: setting,
		Message: prompt,
	}

	resp, err := s.model.GenerateResponse(ctx, req)
	if err != nil {
		return nil, err
	}

	_ = s.logs.Create(ctx, &aiagent.AIUsageLog{
		ID:           uuid.NewString(),
		SchoolID:     input.SchoolID,
		AccountID:    input.AccountID,
		Role:         input.Role,
		Feature:      "question_generation",
		Model:        setting.Model,
		PromptTokens: resp.PromptTokens,
		ResultTokens: resp.ResultTokens,
		TotalTokens:  resp.PromptTokens + resp.ResultTokens,
		CreatedAt:    time.Now(),
	})

	var result GenerateQuestionsResult
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &result, nil
}

func (s *AIGradingUsecase) getAISetting(ctx context.Context, schoolID string) (*aiagent.AIAgentSetting, error) {
	// 尝试查找该学校的配置，必要时使用默认/全局配置。
	// 目前按 SchoolID 查询。
	// 如果系统有专门的“批改 Agent”配置，可以按特定 key 或类型查询。
	// 这里仅获取该学校第一条可用配置或默认配置。

	// 注意：可能需要根据 AI 配置的管理方式（如按 AgentType）调整此查询。
	setting, err := s.settings.GetBySchoolID(ctx, schoolID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrAISettingNotConfigured
		}
		return nil, err
	}
	return setting, nil
}

// cleanJSON 去除 AI 可能添加的 markdown 代码块标记。
func cleanJSON(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
	}
	return strings.TrimSpace(content)
}
