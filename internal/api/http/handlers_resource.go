package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"learn-go/internal/service"
	"learn-go/pkg/middleware"
	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// CreateResourceRequest represents the request body for creating a resource.
type CreateResourceRequest struct {
	Title        string   `json:"title" binding:"required"`
	Description  string   `json:"description"`
	DepartmentID *string  `json:"department_id"`
	Tags         []string `json:"tags"`
	FileIDs      []string `json:"file_ids" binding:"required"`
}

// UpdateResourceRequest represents the request body for updating a resource.
type UpdateResourceRequest struct {
	Title        *string  `json:"title"`
	Description  *string  `json:"description"`
	DepartmentID *string  `json:"department_id"`
	Tags         []string `json:"tags"`
}

// AddFileRequest represents the request body for adding a file to a resource.
type AddFileRequest struct {
	FileID string `json:"file_id" binding:"required"`
}

// CreateTeacherResource creates a new resource.
func (h *Handler) CreateTeacherResource(c *gin.Context) {
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	var req CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidRequest, gin.H{"error": err.Error()})
		return
	}

	resource, err := h.resourceService.CreateResource(c.Request.Context(), accountID, schoolID, service.CreateResourceRequest{
		Title:        req.Title,
		Description:  req.Description,
		DepartmentID: req.DepartmentID,
		Tags:         req.Tags,
		FileIDs:      req.FileIDs,
	})

	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to create resource")
		return
	}

	// Parse tags
	var tags []string
	if resource.Tags != "" {
		_ = json.Unmarshal([]byte(resource.Tags), &tags)
	}

	response.Success(c, http.StatusCreated, gin.H{
		"resource": gin.H{
			"id":              resource.ID,
			"title":           resource.Title,
			"description":     resource.Description,
			"department_id":   resource.DepartmentID,
			"department_name": resource.DepartmentName,
			"grade_level":     resource.GradeLevel,
			"teacher_id":      resource.TeacherID,
			"teacher_name":    resource.TeacherName,
			"tags":            tags,
			"file_count":      resource.FileCount,
			"favorite_count":  resource.FavoriteCount,
			"view_count":      resource.ViewCount,
			"download_count":  resource.DownloadCount,
			"is_favorited":    resource.IsFavorited,
			"created_at":      resource.CreatedAt,
			"updated_at":      resource.UpdatedAt,
		},
	})
}

// ListTeacherResources lists resources created by the teacher.
func (h *Handler) ListTeacherResources(c *gin.Context) {
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	departmentID := c.Query("department_id")
	favoritedOnly := c.Query("favorited_only") == "true"

	resources, total, err := h.resourceService.ListTeacherResources(
		c.Request.Context(),
		accountID,
		schoolID,
		service.ResourceListParams{
			DepartmentID:  departmentID,
			FavoritedOnly: favoritedOnly,
			Page:          page,
			Size:          size,
		},
	)

	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to list resources")
		return
	}

	// Convert resources to response format with parsed tags
	resourcesResponse := make([]gin.H, len(resources))
	for i, resource := range resources {
		var tags []string
		if resource.Tags != "" {
			_ = json.Unmarshal([]byte(resource.Tags), &tags)
		}
		resourcesResponse[i] = gin.H{
			"id":              resource.ID,
			"title":           resource.Title,
			"description":     resource.Description,
			"department_id":   resource.DepartmentID,
			"department_name": resource.DepartmentName,
			"grade_level":     resource.GradeLevel,
			"teacher_id":      resource.TeacherID,
			"teacher_name":    resource.TeacherName,
			"tags":            tags,
			"file_count":      resource.FileCount,
			"favorite_count":  resource.FavoriteCount,
			"view_count":      resource.ViewCount,
			"download_count":  resource.DownloadCount,
			"is_favorited":    resource.IsFavorited,
			"created_at":      resource.CreatedAt,
			"updated_at":      resource.UpdatedAt,
		}
	}

	response.Success(c, http.StatusOK, gin.H{
		"items": resourcesResponse,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetTeacherResource gets a resource detail.
func (h *Handler) GetTeacherResource(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	resource, files, err := h.resourceService.GetResourceDetail(c.Request.Context(), resourceID, accountID, schoolID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to get resource")
		return
	}

	// Parse tags
	var tags []string
	if resource.Tags != "" {
		_ = json.Unmarshal([]byte(resource.Tags), &tags)
	}

	// Convert files to response format
	filesResponse := make([]gin.H, len(files))
	for i, file := range files {
		filesResponse[i] = gin.H{
			"id":          file.ID,
			"name":        file.Name,
			"file_type":   file.Type,
			"size":        file.Size,
			"url":         file.URL,
			"uploaded_at": file.CreatedAt,
		}
	}

	response.Success(c, http.StatusOK, gin.H{
		"resource": gin.H{
			"id":             resource.ID,
			"title":          resource.Title,
			"description":    resource.Description,
			"department_id":  resource.DepartmentID,
			"grade_level":    resource.GradeLevel,
			"tags":           tags,
			"view_count":     resource.ViewCount,
			"download_count": resource.DownloadCount,
			"created_at":     resource.CreatedAt,
			"updated_at":     resource.UpdatedAt,
		},
		"files": filesResponse,
	})
}

// UpdateTeacherResource updates a resource.
func (h *Handler) UpdateTeacherResource(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	var req UpdateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidRequest, gin.H{"error": err.Error()})
		return
	}

	resource, err := h.resourceService.UpdateResource(
		c.Request.Context(),
		resourceID,
		accountID,
		schoolID,
		service.UpdateResourceRequest{
			Title:        req.Title,
			Description:  req.Description,
			DepartmentID: req.DepartmentID,
			Tags:         req.Tags,
		},
	)

	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to update resource")
		return
	}

	// Parse tags
	var tags []string
	if resource.Tags != "" {
		_ = json.Unmarshal([]byte(resource.Tags), &tags)
	}

	response.Success(c, http.StatusOK, gin.H{
		"resource": gin.H{
			"id":              resource.ID,
			"title":           resource.Title,
			"description":     resource.Description,
			"department_id":   resource.DepartmentID,
			"department_name": resource.DepartmentName,
			"grade_level":     resource.GradeLevel,
			"teacher_id":      resource.TeacherID,
			"teacher_name":    resource.TeacherName,
			"tags":            tags,
			"file_count":      resource.FileCount,
			"favorite_count":  resource.FavoriteCount,
			"view_count":      resource.ViewCount,
			"download_count":  resource.DownloadCount,
			"is_favorited":    resource.IsFavorited,
			"created_at":      resource.CreatedAt,
			"updated_at":      resource.UpdatedAt,
		},
	})
}

// DeleteTeacherResource deletes a resource.
func (h *Handler) DeleteTeacherResource(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	if err := h.resourceService.DeleteResource(c.Request.Context(), resourceID, accountID, schoolID); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to delete resource")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "resource deleted"})
}

// AddFileToResource adds a file to a resource.
func (h *Handler) AddFileToResource(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	var req AddFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.resourceService.AddFileToResource(c.Request.Context(), resourceID, req.FileID, accountID, schoolID); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to add file")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "file added"})
}

// RemoveFileFromResource removes a file from a resource.
func (h *Handler) RemoveFileFromResource(c *gin.Context) {
	resourceID := c.Param("id")
	fileID := c.Param("fileID")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	if err := h.resourceService.RemoveFileFromResource(c.Request.Context(), resourceID, fileID, accountID, schoolID); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to remove file")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "file removed"})
}

// BrowseResources lists resources with search and filters.
func (h *Handler) BrowseResources(c *gin.Context) {
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	query := c.Query("query")
	departmentID := c.Query("department_id")
	fileType := c.Query("file_type")
	sort := c.DefaultQuery("sort", "latest")
	favoritedOnly := c.Query("favorited_only") == "true"
	myResourcesOnly := c.Query("my_resources_only") == "true"

	resources, total, err := h.resourceService.BrowseResources(
		c.Request.Context(),
		accountID,
		schoolID,
		service.BrowseResourcesParams{
			Query:           query,
			DepartmentID:    departmentID,
			FileType:        fileType,
			Sort:            sort,
			FavoritedOnly:   favoritedOnly,
			MyResourcesOnly: myResourcesOnly,
			Page:            page,
			Size:            size,
		},
	)

	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to browse resources")
		return
	}

	// Convert resources to response format with parsed tags
	resourcesResponse := make([]gin.H, len(resources))
	for i, resource := range resources {
		var tags []string
		if resource.Tags != "" {
			_ = json.Unmarshal([]byte(resource.Tags), &tags)
		}
		resourcesResponse[i] = gin.H{
			"id":              resource.ID,
			"title":           resource.Title,
			"description":     resource.Description,
			"department_id":   resource.DepartmentID,
			"department_name": resource.DepartmentName,
			"grade_level":     resource.GradeLevel,
			"teacher_id":      resource.TeacherID,
			"teacher_name":    resource.TeacherName,
			"tags":            tags,
			"file_count":      resource.FileCount,
			"favorite_count":  resource.FavoriteCount,
			"view_count":      resource.ViewCount,
			"download_count":  resource.DownloadCount,
			"is_favorited":    resource.IsFavorited,
			"created_at":      resource.CreatedAt,
			"updated_at":      resource.UpdatedAt,
		}
	}

	response.Success(c, http.StatusOK, gin.H{
		"items": resourcesResponse,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetResourceDetail gets resource detail.
func (h *Handler) GetResourceDetail(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	resource, files, err := h.resourceService.GetResourceDetail(c.Request.Context(), resourceID, accountID, schoolID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to get resource")
		return
	}

	// Parse tags
	var tags []string
	if resource.Tags != "" {
		_ = json.Unmarshal([]byte(resource.Tags), &tags)
	}

	// Convert files to response format
	filesResponse := make([]gin.H, len(files))
	for i, file := range files {
		filesResponse[i] = gin.H{
			"id":          file.ID,
			"name":        file.Name,
			"file_type":   file.Type,
			"size":        file.Size,
			"url":         file.URL,
			"uploaded_at": file.CreatedAt,
		}
	}

	response.Success(c, http.StatusOK, gin.H{
		"resource": gin.H{
			"id":             resource.ID,
			"title":          resource.Title,
			"description":    resource.Description,
			"department_id":  resource.DepartmentID,
			"grade_level":    resource.GradeLevel,
			"tags":           tags,
			"view_count":     resource.ViewCount,
			"download_count": resource.DownloadCount,
			"is_favorited":   resource.IsFavorited,
			"created_at":     resource.CreatedAt,
			"updated_at":     resource.UpdatedAt,
		},
		"files": filesResponse,
	})
}

// ToggleFavorite toggles favorite status.
func (h *Handler) ToggleFavorite(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	favorited, err := h.resourceService.ToggleFavorite(c.Request.Context(), resourceID, accountID, schoolID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to toggle favorite")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"favorited": favorited})
}

// ListFavorites lists favorited resources.
func (h *Handler) ListFavorites(c *gin.Context) {
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	resources, total, err := h.resourceService.ListFavorites(c.Request.Context(), accountID, schoolID, page, size)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to list favorites")
		return
	}

	// Convert resources to response format with parsed tags
	resourcesResponse := make([]gin.H, len(resources))
	for i, resource := range resources {
		var tags []string
		if resource.Tags != "" {
			_ = json.Unmarshal([]byte(resource.Tags), &tags)
		}
		resourcesResponse[i] = gin.H{
			"id":              resource.ID,
			"title":           resource.Title,
			"description":     resource.Description,
			"department_id":   resource.DepartmentID,
			"department_name": resource.DepartmentName,
			"grade_level":     resource.GradeLevel,
			"teacher_id":      resource.TeacherID,
			"teacher_name":    resource.TeacherName,
			"tags":            tags,
			"file_count":      resource.FileCount,
			"favorite_count":  resource.FavoriteCount,
			"view_count":      resource.ViewCount,
			"download_count":  resource.DownloadCount,
			"is_favorited":    resource.IsFavorited,
			"created_at":      resource.CreatedAt,
			"updated_at":      resource.UpdatedAt,
		}
	}

	response.Success(c, http.StatusOK, gin.H{
		"items": resourcesResponse,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// DownloadResourceFile handles resource file download.
func (h *Handler) DownloadResourceFile(c *gin.Context) {
	resourceID := c.Param("id")
	fileID := c.Param("fileID")
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	// Increment download count asynchronously
	go func() {
		_ = h.resourceService.IncrementDownloadCount(c.Request.Context(), resourceID)
	}()

	// Get file download info using fileService
	file, method, url, err := h.fileService.GetDownloadInfo(c.Request.Context(), schoolID, fileID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "file_not_found", gin.H{"error": "file not found"})
		return
	}

	// Return download URL
	response.Success(c, http.StatusOK, gin.H{
		"file_id":      file.ID,
		"file_name":    file.Name,
		"download_url": url,
		"method":       method,
	})
}
