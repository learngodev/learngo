package http

import (
	"net/http"
	"strconv"

	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
)

type CreateClassroomRequest struct {
	SchoolID string `json:"school_id" binding:"required"`
	Location string `json:"location" binding:"required"`
}

type UpdateClassroomRequest struct {
	Location string `json:"location" binding:"required"`
}

func (h *Handler) CreateClassroom(c *gin.Context) {
	var req CreateClassroomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	classroom, err := h.classroom.Create(c.Request.Context(), req.SchoolID, req.Location)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create classroom", err.Error())
		return
	}
	response.Success(c, http.StatusCreated, classroom)
}

func (h *Handler) ListClassrooms(c *gin.Context) {
	schoolID := c.Query("school_id")
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	classrooms, total, err := h.classroom.List(c.Request.Context(), schoolID, page, size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list classrooms", err.Error())
		return
	}
	response.Success(c, http.StatusOK, gin.H{
		"classrooms": classrooms,
		"page":       page,
		"page_size":  size,
		"total":      total,
	})
}

func (h *Handler) UpdateClassroom(c *gin.Context) {
	id := c.Param("id")
	var req UpdateClassroomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	classroom, err := h.classroom.Update(c.Request.Context(), id, req.Location)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update classroom", err.Error())
		return
	}
	response.Success(c, http.StatusOK, classroom)
}

func (h *Handler) DeleteClassroom(c *gin.Context) {
	id := c.Param("id")
	if err := h.classroom.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to delete classroom", err.Error())
		return
	}
	response.Success(c, http.StatusOK, nil)
}
