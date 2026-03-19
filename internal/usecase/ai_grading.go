package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	aibiz "learn-go/internal/biz/ai"
	sharedbiz "learn-go/internal/biz/shared"
	"strings"
	"time"
)

// AIGradingService provides AI-powered assignment checking and grading.
type AIGradingService struct {
	settings aibiz.AIAgentSettingRepository
	logs     aibiz.AIUsageLogRepository
	model    AIChatModel
}

// NewAIGradingService creates a new AIGradingService.
func NewAIGradingService(
	settings aibiz.AIAgentSettingRepository,
	logs aibiz.AIUsageLogRepository,
	model AIChatModel,
) *AIGradingService {
	return &AIGradingService{
		settings: settings,
		logs:     logs,
		model:    model,
	}
}

// CheckAssignmentInput contains data for AI pre-check.
type CheckAssignmentInput struct {
	SchoolID    string
	AccountID   string
	Role        sharedbiz.Role
	Title       string
	Description string
	Content     string
}

// CheckAssignmentResult contains AI feedback for students.
type CheckAssignmentResult struct {
	Issues      []string `json:"issues"`
	Suggestions []string `json:"suggestions"`
	Overall     string   `json:"overall"`
}

// ExplainQuestionInput contains data for AI question explanation.
// It is intended to explain the question like a teacher, without providing the final answer.
type ExplainQuestionInput struct {
	SchoolID     string
	AccountID    string
	Role         sharedbiz.Role
	Title        string
	Prompt       string
	QuestionType string
	Options      []string
	ExtraContext string
}

// ExplainQuestionResult contains AI explanation for a question.
// NOTE: It must not include the final answer.
type ExplainQuestionResult struct {
	Analysis  string   `json:"analysis"`
	Steps     []string `json:"steps"`
	KeyPoints []string `json:"key_points"`
	Pitfalls  []string `json:"pitfalls"`
	Checklist []string `json:"checklist"`
}

// GradeAssignmentInput contains data for AI grading.
type GradeAssignmentInput struct {
	AccountID   string
	Role        sharedbiz.Role
	SchoolID    string
	Title       string
	Description string
	Content     string
	Rubrics     string
}

// GradeAssignmentResult contains AI grading results for teachers.
type GradeAssignmentResult struct {
	Score       int      `json:"score"`
	Summary     string   `json:"summary"`
	Suggestions []string `json:"suggestions"`
	ItemScores  []int    `json:"item_scores"`
}

// CheckAssignment performs a pre-check on the assignment for students.
func (s *AIGradingService) CheckAssignment(ctx context.Context, input CheckAssignmentInput) (*CheckAssignmentResult, error) {
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

	_ = s.logs.Create(ctx, &aibiz.AIUsageLog{
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

// ExplainQuestion explains the question like a teacher (in Simplified Chinese), without giving the final answer.
func (s *AIGradingService) ExplainQuestion(ctx context.Context, input ExplainQuestionInput) (*ExplainQuestionResult, error) {
	setting, err := s.getAISetting(ctx, input.SchoolID)
	if err != nil {
		return nil, err
	}

	optionsText := "(none)"
	if len(input.Options) > 0 {
		// Keep options as plain lines; do not imply any correct option.
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

	_ = s.logs.Create(ctx, &aibiz.AIUsageLog{
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

// GradeAssignment performs AI grading for teachers.
func (s *AIGradingService) GradeAssignment(ctx context.Context, input GradeAssignmentInput) (*GradeAssignmentResult, error) {
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

	_ = s.logs.Create(ctx, &aibiz.AIUsageLog{
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

// GenerateQuestionsInput contains parameters for AI question generation.
type GenerateQuestionsInput struct {
	AccountID  string
	Role       sharedbiz.Role
	SchoolID   string
	Topic      string
	Count      int
	Difficulty string
}

// GeneratedQuestion represents a single question generated by AI.
type GeneratedQuestion struct {
	Type    string   `json:"type"` // choice, essay
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"` // Only for choice
	Answer  string   `json:"answer"`
	Score   float64  `json:"score"`
}

// GenerateQuestionsResult contains the list of generated questions.
type GenerateQuestionsResult struct {
	Questions []GeneratedQuestion `json:"questions"`
}

// GenerateQuestions generates assignment questions based on a topic.
func (s *AIGradingService) GenerateQuestions(ctx context.Context, input GenerateQuestionsInput) (*GenerateQuestionsResult, error) {
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

	_ = s.logs.Create(ctx, &aibiz.AIUsageLog{
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

func (s *AIGradingService) getAISetting(ctx context.Context, schoolID string) (*aibiz.AIAgentSetting, error) {
	// Try to find a setting for the school, or use a default/global one if your logic requires.
	// For now, let's assume we fetch by SchoolID.
	// If your system has a specific "Grading Agent" setting, you might query by a specific key or type.
	// Here we just fetch the first available setting for the school or a default one.

	// Note: You might need to adjust this query based on how you manage AI settings (e.g. by AgentType).
	setting, err := s.settings.GetBySchoolID(ctx, schoolID)
	if err != nil {
		if errors.Is(err, sharedbiz.ErrNotFound) {
			return nil, ErrAISettingNotConfigured
		}
		return nil, err
	}
	return setting, nil
}

// cleanJSON helps strip markdown code blocks if the AI adds them.
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
