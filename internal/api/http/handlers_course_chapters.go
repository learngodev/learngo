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

func (h *Handler) ListStudentCourseChapters(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, response.CodeMissingAccountContext, nil)
		return
	}

	courseID := strings.TrimSpace(c.Param("id"))
	if courseID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeMissingCourseID, nil)
		return
	}

	chapters, err := h.student.ListCourseChapters(c.Request.Context(), accountID, courseID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, response.CodeStudentProfileNotFound, nil)
		case errors.Is(err, service.ErrCourseAccessDenied):
			response.Error(c, http.StatusForbidden, response.CodeCourseAccessDenied, nil)
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

func (h *Handler) GetStudentCourseChapter(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, response.CodeMissingAccountContext, nil)
		return
	}

	courseID := strings.TrimSpace(c.Param("id"))
	chapterID := strings.TrimSpace(c.Param("chapterID"))
	if courseID == "" || chapterID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeMissingCourseChapterID, nil)
		return
	}

	chapter, files, err := h.student.GetCourseChapter(c.Request.Context(), accountID, courseID, chapterID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, response.CodeStudentProfileNotFound, nil)
		case errors.Is(err, service.ErrCourseAccessDenied):
			response.Error(c, http.StatusForbidden, response.CodeCourseAccessDenied, nil)
		case errors.Is(err, repository.ErrNotFound):
			response.Error(c, http.StatusNotFound, response.CodeChapterNotFound, nil)
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
