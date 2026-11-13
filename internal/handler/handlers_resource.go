package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"learn-go/internal/usecase"
	"learn-go/pkg/middleware"
	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
)

// CreateResourceRequest 表示创建资源的请求体。
type CreateResourceRequest struct {
	Title        string   `json:"title" binding:"required"`
	Description  string   `json:"description"`
	DepartmentID *string  `json:"department_id"`
	Tags         []string `json:"tags"`
	FileIDs      []string `json:"file_ids" binding:"required"`
}

// UpdateResourceRequest 表示更新资源的请求体。
type UpdateResourceRequest struct {
	Title        *string  `json:"title"`
	Description  *string  `json:"description"`
	DepartmentID *string  `json:"department_id"`
	Tags         []string `json:"tags"`
}

// AddFileRequest 表示向资源添加文件的请求体。
type AddFileRequest struct {
	FileID string `json:"file_id" binding:"required"`
}

// CreateTeacherResource 创建新资源。
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

	resource, err := h.resourceUsecase.CreateResource(c.Request.Context(), accountID, schoolID, usecase.CreateResourceRequest{
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

	// 解析标签
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

// ListTeacherResources 列出当前教师创建的资源。
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

	resources, total, err := h.resourceUsecase.ListTeacherResources(
		c.Request.Context(),
		accountID,
		schoolID,
		usecase.ResourceListParams{
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

	// 将资源转换为响应格式，并解析标签
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

// GetTeacherResource 获取资源详情。
func (h *Handler) GetTeacherResource(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	resource, files, err := h.resourceUsecase.GetResourceDetail(c.Request.Context(), resourceID, accountID, schoolID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to get resource")
		return
	}

	// 解析标签
	var tags []string
	if resource.Tags != "" {
		_ = json.Unmarshal([]byte(resource.Tags), &tags)
	}

	// 将文件转换为响应格式
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

// UpdateTeacherResource 更新资源。
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

	resource, err := h.resourceUsecase.UpdateResource(
		c.Request.Context(),
		resourceID,
		accountID,
		schoolID,
		usecase.UpdateResourceRequest{
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

	// 解析标签
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

// DeleteTeacherResource 删除资源。
func (h *Handler) DeleteTeacherResource(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	if err := h.resourceUsecase.DeleteResource(c.Request.Context(), resourceID, accountID, schoolID); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to delete resource")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "resource deleted"})
}

// AddFileToResource 向资源添加文件。
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

	if err := h.resourceUsecase.AddFileToResource(c.Request.Context(), resourceID, req.FileID, accountID, schoolID); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to add file")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "file added"})
}

// RemoveFileFromResource 从资源中移除文件。
func (h *Handler) RemoveFileFromResource(c *gin.Context) {
	resourceID := c.Param("id")
	fileID := c.Param("fileID")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	if err := h.resourceUsecase.RemoveFileFromResource(c.Request.Context(), resourceID, fileID, accountID, schoolID); err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to remove file")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"message": "file removed"})
}

// BrowseResources 按搜索与筛选条件列出资源。
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

	resources, total, err := h.resourceUsecase.BrowseResources(
		c.Request.Context(),
		accountID,
		schoolID,
		usecase.BrowseResourcesParams{
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

	// 将资源转换为响应格式，并解析标签
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

// GetResourceDetail 获取资源详情。
func (h *Handler) GetResourceDetail(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	resource, files, err := h.resourceUsecase.GetResourceDetail(c.Request.Context(), resourceID, accountID, schoolID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to get resource")
		return
	}

	// 解析标签
	var tags []string
	if resource.Tags != "" {
		_ = json.Unmarshal([]byte(resource.Tags), &tags)
	}

	// 将文件转换为响应格式
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

// ToggleFavorite 切换收藏状态。
func (h *Handler) ToggleFavorite(c *gin.Context) {
	resourceID := c.Param("id")
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	favorited, err := h.resourceUsecase.ToggleFavorite(c.Request.Context(), resourceID, accountID, schoolID)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to toggle favorite")
		return
	}

	response.Success(c, http.StatusOK, gin.H{"favorited": favorited})
}

// ListFavorites 列出已收藏的资源。
func (h *Handler) ListFavorites(c *gin.Context) {
	accountID := c.GetString(middleware.ContextAccountID)
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	resources, total, err := h.resourceUsecase.ListFavorites(c.Request.Context(), accountID, schoolID, page, size)
	if err != nil {
		h.respondServiceError(c, err, http.StatusInternalServerError, "failed to list favorites")
		return
	}

	// 将资源转换为响应格式，并解析标签
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

// DownloadResourceFile 处理资源文件下载。
func (h *Handler) DownloadResourceFile(c *gin.Context) {
	resourceID := c.Param("id")
	fileID := c.Param("fileID")
	schoolID := getSchoolIDFromRequest(c)
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeSchoolIDRequired, nil)
		return
	}

	// 异步增加下载次数
	go func() {
		_ = h.resourceUsecase.IncrementDownloadCount(c.Request.Context(), resourceID)
	}()

	// 通过 fileUsecase 获取文件下载信息
	file, method, url, err := h.fileUsecase.GetDownloadInfo(c.Request.Context(), schoolID, fileID)
	if err != nil {
		response.Error(c, http.StatusNotFound, "file_not_found", gin.H{"error": "file not found"})
		return
	}

	// 返回下载地址
	response.Success(c, http.StatusOK, gin.H{
		"file_id":      file.ID,
		"file_name":    file.Name,
		"download_url": url,
		"method":       method,
	})
}
