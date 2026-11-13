package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"learn-go/internal/domain/identity"
	"learn-go/internal/usecase"
	"learn-go/pkg/middleware"
	"learn-go/pkg/response"
)

type checkAssignmentRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	Content     string `json:"content" validate:"required"`
}

type explainQuestionRequest struct {
	Title        string   `json:"title" validate:"required"`
	Prompt       string   `json:"prompt" validate:"required"`
	QuestionType string   `json:"question_type"`
	Options      []string `json:"options"`
	ExtraContext string   `json:"extra_context"`
}

// CheckAssignment 处理学生作业的提交前检查。
func (h *Handler) CheckAssignment(c *gin.Context) {
	var req checkAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequestBody)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeValidationError)
		return
	}

	accountID := getAccountID(c)
	role := identity.Role(c.GetString(middleware.ContextRole))
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, response.CodeFailedToResolveSchoolContext)
		return
	}

	// 可选：检查学生是否有权限等。
	// 目前直接调用用例即可。

	input := usecase.CheckAssignmentInput{
		SchoolID:    schoolID,
		AccountID:   accountID,
		Role:        role,
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
	}

	result, err := h.aiGrading.CheckAssignment(c.Request.Context(), input)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "ai check failed")
		return
	}

	response.Success(c, http.StatusOK, result)
}

// ExplainQuestion 处理面向学生的 AI 题目讲解。
func (h *Handler) ExplainQuestion(c *gin.Context) {
	var req explainQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequestBody)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeValidationError)
		return
	}

	accountID := getAccountID(c)
	role := identity.Role(c.GetString(middleware.ContextRole))
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, response.CodeFailedToResolveSchoolContext)
		return
	}

	input := usecase.ExplainQuestionInput{
		SchoolID:     schoolID,
		AccountID:    accountID,
		Role:         role,
		Title:        req.Title,
		Prompt:       req.Prompt,
		QuestionType: req.QuestionType,
		Options:      req.Options,
		ExtraContext: req.ExtraContext,
	}

	result, err := h.aiGrading.ExplainQuestion(c.Request.Context(), input)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "ai explain failed")
		return
	}

	response.Success(c, http.StatusOK, result)
}

type gradeAssignmentRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	Content     string `json:"content" validate:"required"`
	Rubrics     string `json:"rubrics"`
}

// GradeAssignment 处理教师的作业批改请求。
func (h *Handler) GradeAssignment(c *gin.Context) {
	var req gradeAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequestBody)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeValidationError)
		return
	}

	accountID := getAccountID(c)
	role := identity.Role(c.GetString(middleware.ContextRole))
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, response.CodeFailedToResolveSchoolContext)
		return
	}

	input := usecase.GradeAssignmentInput{
		SchoolID:    schoolID,
		AccountID:   accountID,
		Role:        role,
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
		Rubrics:     req.Rubrics,
	}

	result, err := h.aiGrading.GradeAssignment(c.Request.Context(), input)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "ai grading failed")
		return
	}

	response.Success(c, http.StatusOK, result)
}

type generateQuestionsRequest struct {
	Topic      string `json:"topic" validate:"required"`
	Count      int    `json:"count" validate:"required,min=1,max=10"`
	Difficulty string `json:"difficulty"`
}

// GenerateQuestions 处理 AI 出题请求。
func (h *Handler) GenerateQuestions(c *gin.Context) {
	var req generateQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequestBody)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeValidationError)
		return
	}

	accountID := getAccountID(c)
	role := identity.Role(c.GetString(middleware.ContextRole))
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, response.CodeFailedToResolveSchoolContext)
		return
	}

	input := usecase.GenerateQuestionsInput{
		SchoolID:   schoolID,
		AccountID:  accountID,
		Role:       role,
		Topic:      req.Topic,
		Count:      req.Count,
		Difficulty: req.Difficulty,
	}

	result, err := h.aiGrading.GenerateQuestions(c.Request.Context(), input)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "ai generation failed")
		return
	}

	response.Success(c, http.StatusOK, result)
}
