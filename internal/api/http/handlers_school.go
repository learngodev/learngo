package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/pkg/middleware"
	"learn-go/pkg/response"
)

type createTimeSlotRequest struct {
	Name      string `json:"name" binding:"required"`
	StartTime string `json:"start_time" binding:"required"` // HH:mm
	EndTime   string `json:"end_time" binding:"required"`   // HH:mm
	SortOrder int    `json:"sort_order"`
}

type updateTimeSlotRequest struct {
	Name      string `json:"name"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	SortOrder int    `json:"sort_order"`
}

func (h *Handler) ListTimeSlots(c *gin.Context) {
	// Assuming single school for now, or get from context if multi-tenant
	// For now, let's assume we get schoolID from the authenticated user or a default one.
	// Since we don't have multi-tenancy fully visible, I'll assume the user's schoolID.

	schoolID := c.GetString("schoolID") // Middleware should set this
	if schoolID == "" {
		// Fallback or error. For now, let's try to get it from the user claims if available.
		// If not, maybe list all? No, TimeSlot has SchoolID.
		// Let's assume the middleware sets "schoolID".
		// If not, we might need to fetch it from the user.
		user, exists := c.Get("user")
		if exists {
			if u, ok := user.(*domain.Account); ok {
				schoolID = u.SchoolID
			}
		}
	}

	if schoolID == "" {
		accountID := c.GetString(middleware.ContextAccountID)
		if accountID != "" {
			account, err := h.auth.GetAccount(c.Request.Context(), accountID)
			if err == nil {
				schoolID = account.SchoolID
				c.Set("user", account)
				c.Set("schoolID", schoolID)
			}
		}
	}

	// Fallback: if schoolID is still empty (e.g. legacy data), try to use the first available school
	if schoolID == "" {
		schools, err := h.school.ListSchools(c.Request.Context())
		if err == nil && len(schools) > 0 {
			schoolID = schools[0].ID
		}
	}

	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	slots, err := h.school.ListTimeSlots(c.Request.Context(), schoolID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_list_time_slots")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"time_slots": slots})
}

func (h *Handler) CreateTimeSlot(c *gin.Context) {
	var req createTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequest)
		return
	}

	schoolID := c.GetString("schoolID")
	if schoolID == "" {
		user, exists := c.Get("user")
		if exists {
			if u, ok := user.(*domain.Account); ok {
				schoolID = u.SchoolID
			}
		}
	}

	if schoolID == "" {
		accountID := c.GetString(middleware.ContextAccountID)
		if accountID != "" {
			account, err := h.auth.GetAccount(c.Request.Context(), accountID)
			if err == nil {
				schoolID = account.SchoolID
				c.Set("user", account)
				c.Set("schoolID", schoolID)
			}
		}
	}

	// Fallback: if schoolID is still empty (e.g. legacy data), try to use the first available school
	if schoolID == "" {
		schools, err := h.school.ListSchools(c.Request.Context())
		if err == nil && len(schools) > 0 {
			schoolID = schools[0].ID
		}
	}

	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	slot := &domain.TimeSlot{
		ID:        uuid.New().String(),
		SchoolID:  schoolID,
		Name:      req.Name,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		SortOrder: req.SortOrder,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.school.CreateTimeSlot(c.Request.Context(), slot); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_create_time_slot")
		return
	}

	response.Success(c, http.StatusCreated, slot)
}

func (h *Handler) UpdateTimeSlot(c *gin.Context) {
	id := c.Param("id")
	var req updateTimeSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequest)
		return
	}

	// Fetch existing slot to ensure it exists and preserve SchoolID
	slot, err := h.school.GetTimeSlot(c.Request.Context(), id)
	if err != nil {
		h.respondServiceError(c, err, http.StatusNotFound, response.CodeTimeSlotNotFound)
		return
	}

	slot.Name = req.Name
	slot.StartTime = req.StartTime
	slot.EndTime = req.EndTime
	slot.SortOrder = req.SortOrder
	slot.UpdatedAt = time.Now()

	if err := h.school.UpdateTimeSlot(c.Request.Context(), slot); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_update_time_slot")
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) DeleteTimeSlot(c *gin.Context) {
	id := c.Param("id")
	if err := h.school.DeleteTimeSlot(c.Request.Context(), id); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_delete_time_slot")
		return
	}

	response.Success(c, http.StatusOK, nil)
}
