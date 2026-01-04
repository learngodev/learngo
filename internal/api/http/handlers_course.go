package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"learn-go/internal/domain"
	"learn-go/internal/service"
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
		response.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	schoolID := h.getSchoolID(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id_required", nil)
		return
	}

	var teacherIDs []string
	accountID := getAccountID(c)
	if teacher, err := h.teacher.GetProfile(c.Request.Context(), accountID); err == nil && teacher != nil {
		teacherIDs = append(teacherIDs, teacher.ID)
	}

	imageURL := ""
	if req.ImageURL != nil {
		imageURL = strings.TrimSpace(*req.ImageURL)
	}

	course, err := h.courseService.CreateCourse(
		c.Request.Context(),
		schoolID,
		teacherIDs,
		req.Name,
		req.Description,
		imageURL,
		req.ClassIDs,
	)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed_to_create_course", err.Error())
		return
	}

	response.Success(c, http.StatusOK, course)
}

func (h *Handler) JoinCourse(c *gin.Context) {
	var req joinCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	accountID := getAccountID(c)
	student, err := h.student.GetStudentByAccountID(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusForbidden, "student_profile_not_found", err.Error())
		return
	}

	if err := h.courseService.JoinCourseByCode(c.Request.Context(), student.ID, req.Code); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed_to_join_course", err.Error())
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) ListStudentCourses(c *gin.Context) {
	accountID := getAccountID(c)
	student, err := h.student.GetStudentByAccountID(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusForbidden, "student_profile_not_found", err.Error())
		return
	}

	courses, err := h.courseService.ListStudentCourses(c.Request.Context(), student.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed_to_list_courses", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"items": courses,
	})
}

func (h *Handler) ListCourses(c *gin.Context) {
	schoolID := h.getSchoolID(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id_required", nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	departmentID := c.Query("department_id")
	classID := c.Query("class_id")

	// If filters are present or we want enriched data, use ListCourseAssignments
	// Actually, the frontend will likely always want the enriched data now.
	// But let's check if we should replace the existing response structure.
	// The existing response is `items: []Course`.
	// The new response is `items: []CourseAssignmentInfo`.
	// This is a breaking change for clients expecting `Course`.
	// However, `CourseAssignmentInfo` contains `course_id`, `course_name`, `description`.
	// It is compatible-ish if the client just looks for those fields, but the field names changed (`id` -> `course_id`, `name` -> `course_name`).
	// To avoid breaking changes, I should probably map it back or use a new endpoint.
	// But since I am the only developer and I am updating the frontend too, I can change the API.
	// Or I can keep `ListCourses` for simple listing and add `ListCourseAssignments` for the management view.
	// The user asked to "improve Course Management info".
	// So I will update `ListCourses` to return the enriched info.
	// I will update the frontend to handle the new structure.

	courses, total, err := h.courseService.ListCoursesWithDetails(c.Request.Context(), schoolID, departmentID, classID, page, size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed_to_list_courses", err.Error())
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
		response.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := h.courseService.UpdateCourseFields(c.Request.Context(), id, req.Name, req.Description, req.ImageURL); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed_to_update_course", err.Error())
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) ListAssignments(c *gin.Context) {
	schoolID := h.getSchoolID(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id_required", nil)
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
		response.Error(c, http.StatusInternalServerError, "failed_to_list_assignments", err.Error())
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
		response.Error(c, http.StatusInternalServerError, "failed_to_delete_course", err.Error())
		return
	}

	response.Success(c, http.StatusOK, nil)
}

type assignStudentsRequest struct {
	ClassID      string `json:"class_id"`
	DepartmentID string `json:"department_id"`
}

type assignCourseClassRequest struct {
	ClassID string `json:"class_id" binding:"required"`
}

func (h *Handler) AssignStudents(c *gin.Context) {
	courseID := c.Param("id")
	var req assignStudentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
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
		response.Error(c, http.StatusInternalServerError, "failed_to_assign_students", err.Error())
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) UpdateTeacherCourse(c *gin.Context) {
	var req updateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	accountID := getAccountID(c)
	courseID := c.Param("id")

	if req.Name == nil && req.Description == nil && req.ImageURL == nil {
		response.Error(c, http.StatusBadRequest, "no_fields_to_update", nil)
		return
	}

	updated, err := h.teacher.UpdateCourse(c.Request.Context(), accountID, courseID, req.Name, req.Description, req.ImageURL)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusForbidden, "teacher_profile_not_found", nil)
		case errors.Is(err, service.ErrTeacherAssignmentForbidden):
			response.Error(c, http.StatusForbidden, "course_not_owned", nil)
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, http.StatusNotFound, "course_not_found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "failed_to_update_course", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"course": updated})
}

func (h *Handler) AssignTeacherCourseClass(c *gin.Context) {
	var req assignCourseClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	accountID := getAccountID(c)
	courseID := c.Param("id")

	if err := h.teacher.AssignCourseClass(c.Request.Context(), accountID, courseID, req.ClassID); err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusForbidden, "teacher_profile_not_found", nil)
		case errors.Is(err, service.ErrTeacherAssignmentForbidden):
			response.Error(c, http.StatusForbidden, "course_not_owned", nil)
		case errors.Is(err, service.ErrInvalidCourseClass):
			response.Error(c, http.StatusBadRequest, "invalid_course_class", nil)
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, http.StatusNotFound, "course_not_found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "failed_to_assign_class", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) RemoveTeacherCourseClass(c *gin.Context) {
	accountID := getAccountID(c)
	courseID := c.Param("id")
	classID := c.Param("classID")

	if err := h.teacher.RemoveCourseClass(c.Request.Context(), accountID, courseID, classID); err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusForbidden, "teacher_profile_not_found", nil)
		case errors.Is(err, service.ErrTeacherAssignmentForbidden):
			response.Error(c, http.StatusForbidden, "course_not_owned", nil)
		case errors.Is(err, service.ErrInvalidCourseClass):
			response.Error(c, http.StatusBadRequest, "invalid_course_class", nil)
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, http.StatusNotFound, "course_not_found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "failed_to_remove_class", err.Error())
		}
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
