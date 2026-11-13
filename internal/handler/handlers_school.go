package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"learn-go/internal/domain/identity"
	"learn-go/internal/domain/scheduling"
	"learn-go/pkg/middleware"
	"learn-go/pkg/response"
)

type createTimeSlotRequest struct {
	Name      string `json:"name" binding:"required"`
	StartTime string `json:"start_time" binding:"required"` // 格式 HH:mm
	EndTime   string `json:"end_time" binding:"required"`   // 格式 HH:mm
	SortOrder int    `json:"sort_order"`
}

type updateTimeSlotRequest struct {
	Name      string `json:"name"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	SortOrder int    `json:"sort_order"`
}

func (h *Handler) ListTimeSlots(c *gin.Context) {
	// 目前假定为单学校；多租户时应从 context 获取
	// 目前假定 schoolID 来自已认证用户或默认学校。
	// 由于多租户尚未完全落地，这里使用当前用户的 schoolID。

	schoolID := c.GetString("schoolID") // 应由中间件设置
	if schoolID == "" {
		// 回退或报错。目前先尝试从用户 claims 中获取。
		// 若获取不到，是否列出全部？不行，TimeSlot 带有 SchoolID。
		// 假定中间件会设置 "schoolID"。
		// 若未设置，可能需要从用户信息中查询。
		user, exists := c.Get("user")
		if exists {
			if u, ok := user.(*identity.Account); ok {
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

	// 回退：若 schoolID 仍为空（如历史数据），尝试使用第一个可用的学校
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
			if u, ok := user.(*identity.Account); ok {
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

	// 回退：若 schoolID 仍为空（如历史数据），尝试使用第一个可用的学校
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

	slot := &scheduling.TimeSlot{
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

	// 查询已有节次，确认其存在并保留 SchoolID
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
