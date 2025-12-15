package http

import (
	"net/http"
	"time"

	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// CreateScheduleRequest defines payload for creating a schedule rule.
type CreateScheduleRequest struct {
	SchoolID  string    `json:"school_id" validate:"required"`
	CourseID  string    `json:"course_id" validate:"required"`
	ClassID   string    `json:"class_id" validate:"required"`
	TeacherID string    `json:"teacher_id" validate:"required"`
	SlotID    string    `json:"slot_id" validate:"required"`
	DayOfWeek int       `json:"day_of_week" validate:"required,min=1,max=7"`
	Location  string    `json:"location"`
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date" validate:"required"`
}

func (h *Handler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	schedule, err := h.schedule.CreateSchedule(c.Request.Context(), req.SchoolID, req.CourseID, req.ClassID, req.TeacherID, req.SlotID, req.DayOfWeek, req.Location, req.StartDate, req.EndDate)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create schedule", err.Error())
		return
	}
	response.Success(c, http.StatusCreated, schedule)
}

// GenerateSessionsRequest defines payload for generating sessions.
type GenerateSessionsRequest struct {
	SchoolID string    `json:"school_id" validate:"required"`
	Start    time.Time `json:"start" validate:"required"`
	End      time.Time `json:"end" validate:"required"`
}

func (h *Handler) GenerateSessions(c *gin.Context) {
	var req GenerateSessionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := h.schedule.GenerateSessions(c.Request.Context(), req.SchoolID, req.Start, req.End); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate sessions", err.Error())
		return
	}
	response.Success(c, http.StatusOK, nil)
}

// UpdateSessionRequest defines payload for updating a session (teacher adjustment).
type UpdateSessionRequest struct {
	SlotID   string    `json:"slot_id" validate:"required"`
	Date     time.Time `json:"date" validate:"required"`
	Location string    `json:"location"`
}

func (h *Handler) UpdateSession(c *gin.Context) {
	id := c.Param("id")
	var req UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := h.schedule.UpdateSession(c.Request.Context(), id, req.SlotID, req.Date, req.Location); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update session", err.Error())
		return
	}
	response.Success(c, http.StatusOK, nil)
}
