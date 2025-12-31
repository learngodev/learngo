package http

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"learn-go/internal/config"
	"learn-go/internal/domain"
	"learn-go/pkg/middleware"
	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UploadRequest struct {
	FileName string `json:"file_name" binding:"required"`
	FileType string `json:"file_type" binding:"required"`
	Size     int64  `json:"size" binding:"required"`
}

// RelayUpload accepts file content (multipart/form-data) and uploads it to OSS via the server.
// Form fields:
// - file: required
// - file_name: optional (fallback to original filename)
// - file_type: optional (fallback to part Content-Type)
func (h *Handler) RelayUpload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "file is required", nil)
		return
	}

	fileID := strings.TrimSpace(c.PostForm("file_id"))

	fileName := strings.TrimSpace(c.PostForm("file_name"))
	if fileName == "" {
		fileName = strings.TrimSpace(fileHeader.Filename)
	}
	fileType := strings.TrimSpace(c.PostForm("file_type"))
	if fileType == "" {
		fileType = strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	}
	if fileType == "" {
		fileType = "application/octet-stream"
	}
	if mediaType, _, err := mime.ParseMediaType(fileType); err == nil && mediaType != "" {
		fileType = mediaType
	}

	size := fileHeader.Size
	if size <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid file size", nil)
		return
	}

	if fileID == "" {
		if fileName == "" {
			response.Error(c, http.StatusBadRequest, "file_name is required", nil)
			return
		}
		if len(fileName) > 256 {
			response.Error(c, http.StatusBadRequest, "file_name too long", nil)
			return
		}
		if len(fileType) > 255 {
			response.Error(c, http.StatusBadRequest, "file_type too long", nil)
			return
		}
	}

	accountID := c.GetString(middleware.ContextAccountID)
	account, err := h.auth.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "account not found", nil)
		return
	}

	opened, err := fileHeader.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to open uploaded file", err.Error())
		return
	}
	defer opened.Close()

	var stored *domain.File
	if fileID != "" {
		recorded, err := h.fileService.GetFileForUploader(
			c.Request.Context(),
			account.SchoolID,
			account.ID,
			fileID,
		)
		if err != nil {
			response.Error(c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		// Best-effort safety: prevent mismatched metadata when reusing an existing record.
		if fileName != "" && len(fileName) > 256 {
			response.Error(c, http.StatusBadRequest, "file_name too long", nil)
			return
		}
		if fileType != "" && len(fileType) > 255 {
			response.Error(c, http.StatusBadRequest, "file_type too long", nil)
			return
		}
		if strings.TrimSpace(fileName) != "" && strings.TrimSpace(fileName) != strings.TrimSpace(recorded.Name) {
			response.Error(c, http.StatusBadRequest, "file_name mismatch", nil)
			return
		}
		if strings.TrimSpace(fileType) != "" {
			if mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(fileType)); err == nil && mediaType != "" {
				fileType = mediaType
			}
			if strings.TrimSpace(fileType) != strings.TrimSpace(recorded.Type) {
				response.Error(c, http.StatusBadRequest, "file_type mismatch", nil)
				return
			}
		}
		if recorded.Size > 0 && size != recorded.Size {
			response.Error(c, http.StatusBadRequest, "file_size mismatch", nil)
			return
		}
		if err := h.fileService.RelayUploadExisting(c.Request.Context(), recorded, opened); err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
			return
		}
		stored = recorded
	} else {
		recorded, err := h.fileService.RelayUpload(
			c.Request.Context(),
			account.SchoolID,
			account.ID,
			fileName,
			fileType,
			size,
			opened,
		)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err.Error(), nil)
			return
		}
		stored = recorded
	}

	response.Success(c, http.StatusOK, gin.H{
		"file_id":   stored.ID,
		"file_name": stored.Name,
		"file_type": stored.Type,
		"size":      stored.Size,
		"key":       stored.Key,
	})
}

func (h *Handler) GetUploadURL(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	req.FileName = strings.TrimSpace(req.FileName)
	if req.FileName == "" {
		response.Error(c, http.StatusBadRequest, "file_name is required", nil)
		return
	}
	if len(req.FileName) > 256 {
		response.Error(c, http.StatusBadRequest, "file_name too long", nil)
		return
	}

	req.FileType = strings.TrimSpace(req.FileType)
	if req.FileType == "" {
		response.Error(c, http.StatusBadRequest, "file_type is required", nil)
		return
	}
	if mediaType, _, err := mime.ParseMediaType(req.FileType); err == nil && mediaType != "" {
		req.FileType = mediaType
	}
	if len(req.FileType) > 255 {
		response.Error(c, http.StatusBadRequest, "file_type too long", nil)
		return
	}

	accountID := c.GetString(middleware.ContextAccountID)
	account, err := h.auth.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "account not found", nil)
		return
	}

	file, method, url, err := h.fileService.InitUpload(
		c.Request.Context(),
		account.SchoolID,
		account.ID,
		req.FileName,
		req.FileType,
		req.Size,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	out := gin.H{
		"file_id":       file.ID,
		"file_name":     file.Name,
		"file_type":     file.Type,
		"size":          file.Size,
		"key":           file.Key,
		"upload_method": method,
	}
	if method == "direct" {
		out["upload_url"] = url
	} else {
		out["relay_url"] = "/api/v1/files/upload/relay"
	}

	response.Success(c, http.StatusOK, out)
}

func (h *Handler) GetDownloadURL(c *gin.Context) {
	fileID := strings.TrimSpace(c.Param("id"))
	if fileID == "" {
		response.Error(c, http.StatusBadRequest, "file id required", nil)
		return
	}

	accountID := c.GetString(middleware.ContextAccountID)
	account, err := h.auth.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "account not found", nil)
		return
	}

	_, method, directURL, err := h.fileService.GetDownloadInfo(c.Request.Context(), account.SchoolID, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "file not found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}

	outURL := directURL
	if strings.EqualFold(method, "relay") {
		cfg := config.Load()
		claims := jwt.MapClaims{
			"sub": account.ID,
			"sid": account.SchoolID,
			"fid": fileID,
			"typ": "download",
			"exp": time.Now().Add(5 * time.Minute).Unix(),
			"iat": time.Now().Unix(),
			"jti": uuid.NewString(),
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := tok.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "failed to sign download token", nil)
			return
		}
		base := requestBaseURL(c)
		outURL = fmt.Sprintf("%s/api/v1/files/download/relay/%s?token=%s", base, url.PathEscape(fileID), url.QueryEscape(signed))
	}

	response.Success(c, http.StatusOK, gin.H{
		"url":             outURL,
		"download_method": strings.ToLower(method),
	})
}

func (h *Handler) RelayDownload(c *gin.Context) {
	fileID := strings.TrimSpace(c.Param("id"))
	if fileID == "" {
		response.Error(c, http.StatusBadRequest, "file id required", nil)
		return
	}

	tokenString := strings.TrimSpace(c.Query("token"))
	if tokenString == "" {
		response.Error(c, http.StatusUnauthorized, "missing token", nil)
		return
	}

	cfg := config.Load()
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		response.Error(c, http.StatusUnauthorized, "invalid token", nil)
		return
	}

	claimType, _ := claims["typ"].(string)
	if claimType != "download" {
		response.Error(c, http.StatusUnauthorized, "invalid token", nil)
		return
	}

	sub, _ := claims["sub"].(string)
	schoolID, _ := claims["sid"].(string)
	claimedFileID, _ := claims["fid"].(string)
	if sub == "" || schoolID == "" || claimedFileID == "" || claimedFileID != fileID {
		response.Error(c, http.StatusUnauthorized, "invalid token", nil)
		return
	}

	account, err := h.auth.GetAccount(c.Request.Context(), sub)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, "account not found", nil)
		return
	}
	if account.SchoolID != schoolID {
		response.Error(c, http.StatusForbidden, "forbidden", nil)
		return
	}

	file, rc, err := h.fileService.OpenDownloadStream(c.Request.Context(), schoolID, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "file not found", nil)
			return
		}
		response.Error(c, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	defer rc.Close()

	contentType := strings.TrimSpace(file.Type)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	name := strings.TrimSpace(file.Name)
	if name == "" {
		name = file.ID
	}

	disposition := "attachment"
	if isInlinePreviewContentType(contentType) {
		disposition = "inline"
	}

	headers := map[string]string{
		"Content-Disposition": fmt.Sprintf("%s; filename=\"%s\"", disposition, sanitizeFilename(name)),
	}

	length := file.Size
	if length <= 0 {
		length = -1
	}
	c.DataFromReader(http.StatusOK, length, contentType, rc, headers)
}

func isInlinePreviewContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	return strings.HasPrefix(ct, "image/")
}

func requestBaseURL(c *gin.Context) string {
	proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}

	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s", proto, host)
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "\r", "_")
	name = strings.ReplaceAll(name, "\n", "_")
	return name
}
