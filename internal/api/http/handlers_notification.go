package http

import (
	"net/http"
	"strconv"

	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	offset := (page - 1) * size

	notifications, total, err := h.notifications.ListByUser(c.Request.Context(), userID, size, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list notifications", err.Error())
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
	userID := c.GetString("userID")
	id := c.Param("id")

	if err := h.notifications.MarkAsRead(c.Request.Context(), id, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to mark notification as read", err.Error())
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) MarkAllNotificationsAsRead(c *gin.Context) {
	userID := c.GetString("userID")

	if err := h.notifications.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to mark all notifications as read", err.Error())
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) CountUnreadNotifications(c *gin.Context) {
	userID := c.GetString("userID")

	count, err := h.notifications.CountUnread(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to count unread notifications", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"count": count})
}
