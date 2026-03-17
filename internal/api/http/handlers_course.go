package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"learn-go/internal/domain"
	"learn-go/pkg/response"
)

type createCourseRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	ImageURL    *string  `json:"image_url"`
	ClassIDs    []string `json:"class_ids"`
}

type updateCourseRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
}

type joinCourseRequest struct {
	Code string `json:"code" binding:"required"`
}

func (h *Handler) CreateCourse(c *gin.Context) {
	var req createCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequest)
		return
	}

	schoolID := h.getSchoolID(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	imageURL := ""
	if req.ImageURL != nil {
		imageURL = strings.TrimSpace(*req.ImageURL)
	}

	course, err := h.courseService.CreateCourse(
		c.Request.Context(),
		schoolID,
		req.Name,
		req.Description,
		imageURL,
		req.ClassIDs,
	)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_create_course")
		return
	}

	response.Success(c, http.StatusOK, course)
}

func (h *Handler) JoinCourse(c *gin.Context) {
	var req joinCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequest)
		return
	}

	accountID := getAccountID(c)
	student, err := h.student.GetStudentByAccountID(c.Request.Context(), accountID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusForbidden, response.CodeStudentProfileNotFound)
		return
	}

	if err := h.courseService.JoinCourseByCode(c.Request.Context(), student.ID, req.Code); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_join_course")
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) ListStudentCourses(c *gin.Context) {
	accountID := getAccountID(c)
	student, err := h.student.GetStudentByAccountID(c.Request.Context(), accountID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusForbidden, response.CodeStudentProfileNotFound)
		return
	}

	courses, err := h.courseService.ListStudentCourses(c.Request.Context(), student.ID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_list_courses")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"items": courses,
	})
}

func (h *Handler) ListCourses(c *gin.Context) {
	schoolID := h.getSchoolID(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	departmentID := c.Query("department_id")
	classID := c.Query("class_id")

	courses, total, err := h.courseService.ListCoursesWithDetails(c.Request.Context(), schoolID, departmentID, classID, page, size)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_list_courses")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"items": courses,
		"total": total,
	})
}

func (h *Handler) UpdateCourse(c *gin.Context) {
	id := c.Param("id")
	var req updateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequest)
		return
	}

	if err := h.courseService.UpdateCourseFields(c.Request.Context(), id, req.Name, req.Description, req.ImageURL); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_update_course")
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) ListAssignments(c *gin.Context) {
	schoolID := h.getSchoolID(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	courseID := c.Query("course_id")
	departmentID := c.Query("department_id")
	classID := c.Query("class_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	// Use ListCourseAssignments which now queries course_schedules
	assignments, total, err := h.courseService.ListCourseAssignments(c.Request.Context(), schoolID, courseID, departmentID, classID, true, page, size)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_list_assignments")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"items": assignments,
		"total": total,
	})
}

func (h *Handler) DeleteCourse(c *gin.Context) {
	id := c.Param("id")
	if err := h.courseService.DeleteCourse(c.Request.Context(), id); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_delete_course")
		return
	}

	response.Success(c, http.StatusOK, nil)
}

type assignStudentsRequest struct {
	ClassID      string `json:"class_id"`
	DepartmentID string `json:"department_id"`
}

func (h *Handler) AssignStudents(c *gin.Context) {
	courseID := c.Param("id")
	var req assignStudentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respondServiceError(c, err, http.StatusBadRequest, response.CodeInvalidRequest)
		return
	}

	ctx := c.Request.Context()
	var err error

	if req.ClassID != "" {
		err = h.courseService.AssignStudentsByClass(ctx, courseID, req.ClassID)
	} else if req.DepartmentID != "" {
		err = h.courseService.AssignStudentsByDepartment(ctx, courseID, req.DepartmentID)
	} else {
		response.Error(c, http.StatusBadRequest, "missing_assignment_target", nil)
		return
	}

	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed_to_assign_students")
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) getSchoolID(c *gin.Context) string {
	// First try query param
	if sid := c.Query("school_id"); sid != "" {
		return sid
	}

	// Then try context (middleware)
	schoolID := c.GetString("schoolID")
	if schoolID != "" {
		return schoolID
	}

	user, exists := c.Get("user")
	if exists {
		if u, ok := user.(*domain.Account); ok {
			return u.SchoolID
		}
	}
	return ""
}
