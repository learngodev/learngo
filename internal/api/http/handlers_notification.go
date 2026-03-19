package http

import (
	"net/http"
	"strconv"

	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListNotifications(c *gin.Context) {
	userID := getAccountID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, response.CodeMissingAccountContext, nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	offset := (page - 1) * size

	notifications, total, err := h.notifications.ListByUser(c.Request.Context(), userID, size, offset)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "Failed to list notifications")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"items": notifications,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func (h *Handler) MarkNotificationAsRead(c *gin.Context) {
	userID := getAccountID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, response.CodeMissingAccountContext, nil)
		return
	}
	id := c.Param("id")

	if err := h.notifications.MarkAsRead(c.Request.Context(), id, userID); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "Failed to mark notification as read")
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) MarkAllNotificationsAsRead(c *gin.Context) {
	userID := getAccountID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, response.CodeMissingAccountContext, nil)
		return
	}

	if err := h.notifications.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "Failed to mark all notifications as read")
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) CountUnreadNotifications(c *gin.Context) {
	userID := getAccountID(c)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, response.CodeMissingAccountContext, nil)
		return
	}

	count, err := h.notifications.CountUnread(c.Request.Context(), userID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "Failed to count unread notifications")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"count": count})
}
