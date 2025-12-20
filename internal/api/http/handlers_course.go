package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"learn-go/internal/domain"
	"learn-go/pkg/response"
)

type createCourseRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type updateCourseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type assignCourseRequest struct {
	CourseID  string `json:"course_id" binding:"required"`
	TeacherID string `json:"teacher_id" binding:"required"`
	ClassID   string `json:"class_id" binding:"required"`
}

type updateTeachingAssignmentRequest struct {
	TeacherID string `json:"teacher_id" binding:"required"`
	ClassID   string `json:"class_id" binding:"required"`
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

	course, err := h.courseService.CreateCourse(c.Request.Context(), schoolID, req.Name, req.Description)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed_to_create_course", err.Error())
		return
	}

	response.Success(c, http.StatusOK, course)
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

	if err := h.courseService.UpdateCourse(c.Request.Context(), id, req.Name, req.Description); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed_to_update_course", err.Error())
		return
	}

	response.Success(c, http.StatusOK, nil)
}

type classAssignmentOverview struct {
	ID            string   `json:"id"`
	ClassID       string   `json:"class_id"`
	ClassName     string   `json:"class_name"`
	StudentCount  int64    `json:"student_count"`
	AssignmentIDs []string `json:"assignment_ids"`
	TeacherNames  []string `json:"teacher_names"`
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
