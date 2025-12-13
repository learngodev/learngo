package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"learn-go/internal/service"
	"learn-go/pkg/response"
)

type checkAssignmentRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	Content     string `json:"content" validate:"required"`
}

// CheckAssignment handles student assignment pre-check.
func (h *Handler) CheckAssignment(c *gin.Context) {
	var req checkAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	accountID := getAccountID(c)
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to resolve school context", err.Error())
		return
	}

	// Optional: Check if student has permission, etc.
	// For now, we just use the service.

	input := service.CheckAssignmentInput{
		SchoolID:    schoolID,
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
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	accountID := getAccountID(c)
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to resolve school context", err.Error())
		return
	}

	input := service.GradeAssignmentInput{
		SchoolID:    schoolID,
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
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	accountID := getAccountID(c)
	schoolID, err := h.teacher.GetSchoolID(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to resolve school context", err.Error())
		return
	}

	input := service.GenerateQuestionsInput{
		SchoolID:   schoolID,
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
