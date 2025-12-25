package http

import (
	"net/http"

	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type UploadRequest struct {
	FileName string `json:"file_name" binding:"required"`
	FileType string `json:"file_type" binding:"required"`
	Size     int64  `json:"size" binding:"required"`
}

func (h *Handler) GetUploadURL(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	schoolID := c.GetString("school_id")
	userID := c.GetString("user_id")

	file, url, err := h.fileService.GetUploadURL(c.Request.Context(), schoolID, userID, req.FileName, req.FileType, req.Size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"file_id":    file.ID,
		"upload_url": url,
		"key":        file.Key,
	})
}

func (h *Handler) GetDownloadURL(c *gin.Context) {
	fileID := c.Param("id")
	url, err := h.fileService.GetDownloadURL(c.Request.Context(), fileID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"url": url,
	})
}
