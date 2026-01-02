package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"learn-go/internal/repository"
	"learn-go/internal/service"
	"learn-go/pkg/response"
)

type createTeacherCourseChapterRequest struct {
	Title      string `json:"title" validate:"required"`
	Content    string `json:"content"`
	OrderIndex int    `json:"order_index"`
}

type updateTeacherCourseChapterRequest struct {
	Title      *string `json:"title"`
	Content    *string `json:"content"`
	OrderIndex *int    `json:"order_index"`
}

type attachTeacherCourseChapterFileRequest struct {
	FileID string `json:"file_id" validate:"required"`
}

func (h *Handler) ListTeacherCourseChapters(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	courseID := strings.TrimSpace(c.Param("id"))
	if courseID == "" {
		response.Error(c, http.StatusBadRequest, "missing course id", nil)
		return
	}

	chapters, err := h.teacher.ListCourseChapters(c.Request.Context(), accountID, courseID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		case errors.Is(err, service.ErrTeacherCourseAccessDenied):
			response.Error(c, http.StatusForbidden, "course access denied", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to list chapters", err.Error())
		}
		return
	}

	payload := make([]gin.H, 0, len(chapters))
	for _, chapter := range chapters {
		payload = append(payload, gin.H{
			"id":          chapter.ID,
			"course_id":   chapter.CourseID,
			"teacher_id":  chapter.TeacherID,
			"title":       chapter.Title,
			"order_index": chapter.OrderIndex,
			"created_at":  chapter.CreatedAt,
			"updated_at":  chapter.UpdatedAt,
		})
	}

	response.Success(c, http.StatusOK, gin.H{"items": payload})
}

func (h *Handler) GetTeacherCourseChapter(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	courseID := strings.TrimSpace(c.Param("id"))
	chapterID := strings.TrimSpace(c.Param("chapterID"))
	if courseID == "" || chapterID == "" {
		response.Error(c, http.StatusBadRequest, "missing course/chapter id", nil)
		return
	}

	chapter, files, err := h.teacher.GetCourseChapter(c.Request.Context(), accountID, courseID, chapterID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		case errors.Is(err, service.ErrTeacherCourseAccessDenied):
			response.Error(c, http.StatusForbidden, "course access denied", nil)
		case errors.Is(err, repository.ErrNotFound):
			response.Error(c, http.StatusNotFound, "chapter not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to get chapter", err.Error())
		}
		return
	}

	attachments := make([]gin.H, 0, len(files))
	for _, f := range files {
		attachments = append(attachments, gin.H{
			"id":        f.ID,
			"name":      f.Name,
			"type":      f.Type,
			"size":      f.Size,
			"relay_url": fmt.Sprintf("/api/v1/files/download/relay/%s", f.ID),
		})
	}

	response.Success(c, http.StatusOK, gin.H{
		"chapter": gin.H{
			"id":          chapter.ID,
			"course_id":   chapter.CourseID,
			"teacher_id":  chapter.TeacherID,
			"title":       chapter.Title,
			"content":     chapter.Content,
			"order_index": chapter.OrderIndex,
			"created_at":  chapter.CreatedAt,
			"updated_at":  chapter.UpdatedAt,
		},
		"attachments": attachments,
	})
}

func (h *Handler) CreateTeacherCourseChapter(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	courseID := strings.TrimSpace(c.Param("id"))
	if courseID == "" {
		response.Error(c, http.StatusBadRequest, "missing course id", nil)
		return
	}

	var req createTeacherCourseChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	chapter, err := h.teacher.CreateCourseChapter(c.Request.Context(), accountID, courseID, req.Title, req.Content, req.OrderIndex)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		case errors.Is(err, service.ErrTeacherCourseAccessDenied):
			response.Error(c, http.StatusForbidden, "course access denied", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to create chapter", err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"chapter": gin.H{
		"id":          chapter.ID,
		"course_id":   chapter.CourseID,
		"teacher_id":  chapter.TeacherID,
		"title":       chapter.Title,
		"content":     chapter.Content,
		"order_index": chapter.OrderIndex,
		"created_at":  chapter.CreatedAt,
		"updated_at":  chapter.UpdatedAt,
	}})
}

func (h *Handler) UpdateTeacherCourseChapter(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	courseID := strings.TrimSpace(c.Param("id"))
	chapterID := strings.TrimSpace(c.Param("chapterID"))
	if courseID == "" || chapterID == "" {
		response.Error(c, http.StatusBadRequest, "missing course/chapter id", nil)
		return
	}

	var req updateTeacherCourseChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	updates := map[string]any{}
	if req.Title != nil {
		updates["title"] = strings.TrimSpace(*req.Title)
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.OrderIndex != nil {
		updates["order_index"] = *req.OrderIndex
	}
	if len(updates) == 0 {
		response.Error(c, http.StatusBadRequest, "no fields to update", nil)
		return
	}

	chapter, err := h.teacher.UpdateCourseChapter(c.Request.Context(), accountID, courseID, chapterID, updates)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		case errors.Is(err, service.ErrTeacherCourseAccessDenied):
			response.Error(c, http.StatusForbidden, "course access denied", nil)
		case errors.Is(err, repository.ErrNotFound):
			response.Error(c, http.StatusNotFound, "chapter not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to update chapter", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"chapter": gin.H{
		"id":          chapter.ID,
		"course_id":   chapter.CourseID,
		"teacher_id":  chapter.TeacherID,
		"title":       chapter.Title,
		"content":     chapter.Content,
		"order_index": chapter.OrderIndex,
		"created_at":  chapter.CreatedAt,
		"updated_at":  chapter.UpdatedAt,
	}})
}

func (h *Handler) DeleteTeacherCourseChapter(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	courseID := strings.TrimSpace(c.Param("id"))
	chapterID := strings.TrimSpace(c.Param("chapterID"))
	if courseID == "" || chapterID == "" {
		response.Error(c, http.StatusBadRequest, "missing course/chapter id", nil)
		return
	}

	if err := h.teacher.DeleteCourseChapter(c.Request.Context(), accountID, courseID, chapterID); err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		case errors.Is(err, service.ErrTeacherCourseAccessDenied):
			response.Error(c, http.StatusForbidden, "course access denied", nil)
		case errors.Is(err, repository.ErrNotFound):
			response.Error(c, http.StatusNotFound, "chapter not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to delete chapter", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) AttachTeacherCourseChapterFile(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	courseID := strings.TrimSpace(c.Param("id"))
	chapterID := strings.TrimSpace(c.Param("chapterID"))
	if courseID == "" || chapterID == "" {
		response.Error(c, http.StatusBadRequest, "missing course/chapter id", nil)
		return
	}

	var req attachTeacherCourseChapterFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}
	if err := h.validate.Struct(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	if err := h.teacher.AttachCourseChapterFile(c.Request.Context(), accountID, courseID, chapterID, strings.TrimSpace(req.FileID)); err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		case errors.Is(err, service.ErrTeacherCourseAccessDenied):
			response.Error(c, http.StatusForbidden, "course access denied", nil)
		case errors.Is(err, repository.ErrNotFound):
			response.Error(c, http.StatusNotFound, "chapter or file not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to attach file", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"attached": true})
}

func (h *Handler) DetachTeacherCourseChapterFile(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	courseID := strings.TrimSpace(c.Param("id"))
	chapterID := strings.TrimSpace(c.Param("chapterID"))
	fileID := strings.TrimSpace(c.Param("fileID"))
	if courseID == "" || chapterID == "" || fileID == "" {
		response.Error(c, http.StatusBadRequest, "missing course/chapter/file id", nil)
		return
	}

	if err := h.teacher.DetachCourseChapterFile(c.Request.Context(), accountID, courseID, chapterID, fileID); err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		case errors.Is(err, service.ErrTeacherCourseAccessDenied):
			response.Error(c, http.StatusForbidden, "course access denied", nil)
		case errors.Is(err, repository.ErrNotFound):
			response.Error(c, http.StatusNotFound, "chapter not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to detach file", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"detached": true})
}
