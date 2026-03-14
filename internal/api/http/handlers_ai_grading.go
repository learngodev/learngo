package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"learn-go/internal/domain"
	"learn-go/internal/service"
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

// CheckAssignment handles student assignment pre-check.
func (h *Handler) CheckAssignment(c *gin.Context) {
	var req checkAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidRequestBody, err.Error())
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeValidationError, err.Error())
		return
	}

	accountID := getAccountID(c)
	role := domain.Role(c.GetString(middleware.ContextRole))
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeFailedToResolveSchoolContext, err.Error())
		return
	}

	// Optional: Check if student has permission, etc.
	// For now, we just use the service.

	input := service.CheckAssignmentInput{
		SchoolID:    schoolID,
		AccountID:   accountID,
		Role:        role,
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
	}

	result, err := h.aiGrading.CheckAssignment(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ai check failed", err.Error())
		return
	}

	response.Success(c, http.StatusOK, result)
}

// ExplainQuestion handles AI question explanation for students.
func (h *Handler) ExplainQuestion(c *gin.Context) {
	var req explainQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidRequestBody, err.Error())
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeValidationError, err.Error())
		return
	}

	accountID := getAccountID(c)
	role := domain.Role(c.GetString(middleware.ContextRole))
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeFailedToResolveSchoolContext, err.Error())
		return
	}

	input := service.ExplainQuestionInput{
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
		response.Error(c, http.StatusInternalServerError, "ai explain failed", err.Error())
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

// GradeAssignment handles teacher assignment grading.
func (h *Handler) GradeAssignment(c *gin.Context) {
	var req gradeAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidRequestBody, err.Error())
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeValidationError, err.Error())
		return
	}

	accountID := getAccountID(c)
	role := domain.Role(c.GetString(middleware.ContextRole))
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeFailedToResolveSchoolContext, err.Error())
		return
	}

	input := service.GradeAssignmentInput{
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
		response.Error(c, http.StatusInternalServerError, "ai grading failed", err.Error())
		return
	}

	response.Success(c, http.StatusOK, result)
}

type generateQuestionsRequest struct {
	Topic      string `json:"topic" validate:"required"`
	Count      int    `json:"count" validate:"required,min=1,max=10"`
	Difficulty string `json:"difficulty"`
}

// GenerateQuestions handles AI question generation.
func (h *Handler) GenerateQuestions(c *gin.Context) {
	var req generateQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidRequestBody, err.Error())
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeValidationError, err.Error())
		return
	}

	accountID := getAccountID(c)
	role := domain.Role(c.GetString(middleware.ContextRole))
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeFailedToResolveSchoolContext, err.Error())
		return
	}

	input := service.GenerateQuestionsInput{
		SchoolID:   schoolID,
		AccountID:  accountID,
		Role:       role,
		Topic:      req.Topic,
		Count:      req.Count,
		Difficulty: req.Difficulty,
	}

	result, err := h.aiGrading.GenerateQuestions(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "ai generation failed", err.Error())
		return
	}

	response.Success(c, http.StatusOK, result)
}
