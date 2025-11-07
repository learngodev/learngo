package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"learn-go/internal/api/grpcmapper"
	"learn-go/internal/api/grpcpb"
	"learn-go/internal/domain"
	"learn-go/internal/realtime"
	"learn-go/internal/service"
	"learn-go/pkg/middleware"
	"learn-go/pkg/response"
)

// Handler aggregates dependencies for HTTP handlers.
type Handler struct {
	auth          *service.AuthService
	admin         *service.AdminService
	assignments   *service.AssignmentService
	notes         *service.NoteService
	noteComments  *service.NoteCommentService
	conversations *service.ConversationService
	oss           *service.AdminOssService
	system        *service.AdminSystemService
	streamHub     *realtime.Hub
	validate      *validator.Validate
}

// NewHandler constructs a Handler instance.
func NewHandler(auth *service.AuthService, admin *service.AdminService, assignments *service.AssignmentService, conversations *service.ConversationService, notes *service.NoteService, noteComments *service.NoteCommentService, oss *service.AdminOssService, system *service.AdminSystemService, streamHub *realtime.Hub) *Handler {
	return &Handler{
		auth:          auth,
		admin:         admin,
		assignments:   assignments,
		notes:         notes,
		noteComments:  noteComments,
		conversations: conversations,
		oss:           oss,
		system:        system,
		streamHub:     streamHub,
		validate:      validator.New(),
	}
}

// RegisterRoutes attaches HTTP endpoints to router.
func (h *Handler) RegisterRoutes(r *gin.Engine, adminGuard gin.HandlerFunc, teacherGuard gin.HandlerFunc, studentGuard gin.HandlerFunc) {
	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", h.Login)

		admin := api.Group("/admin", adminGuard)
		admin.POST("/teachers", h.CreateTeacher)
		admin.POST("/students", h.CreateStudent)
		admin.POST("/departments", h.CreateDepartment)
		admin.PATCH("/departments/:id", h.UpdateDepartment)
		admin.DELETE("/departments/:id", h.DeleteDepartment)
		admin.POST("/classes", h.CreateClass)
		admin.PATCH("/classes/:id", h.UpdateClass)
		admin.DELETE("/classes/:id", h.DeleteClass)
		admin.GET("/departments", h.ListDepartments)
		admin.GET("/departments/:id/classes", h.ListClasses)
		admin.GET("/accounts", h.ListAccounts)
		admin.POST("/accounts/:id/password/reset", h.ResetAccountPassword)
		admin.POST("/accounts/:id/lock", h.LockAccount)
		admin.POST("/accounts/:id/unlock", h.UnlockAccount)
		admin.DELETE("/accounts/:id", h.DeleteAccount)
		admin.POST("/oss/credentials", h.CreateOssCredential)
		admin.GET("/oss/credentials", h.ListOssCredentials)
		admin.PATCH("/oss/credentials/:id", h.UpdateOssCredential)
		admin.DELETE("/oss/credentials/:id", h.DeleteOssCredential)
		admin.POST("/oss/policies", h.CreateOssPolicy)
		admin.GET("/oss/policies", h.ListOssPolicies)
		admin.PATCH("/oss/policies/:id", h.UpdateOssPolicy)
		admin.DELETE("/oss/policies/:id", h.DeleteOssPolicy)
		admin.GET("/oss/audit_logs", h.ListOssAuditLogs)
		admin.GET("/system/switches", h.ListSystemSwitches)
		admin.PATCH("/system/switches/:id", h.UpdateSystemSwitch)
		admin.GET("/system/parameters", h.ListSystemParameters)
		admin.PATCH("/system/parameters/:id", h.UpdateSystemParameter)
		admin.GET("/system/broadcasts", h.ListSystemBroadcasts)
		admin.PATCH("/system/broadcasts/:id", h.UpdateSystemBroadcast)
		admin.GET("/system/audit_logs", h.ListSystemAuditLogs)

		assignments := api.Group("/assignments", teacherGuard)
		assignments.POST("", h.CreateAssignment)
		assignments.GET(":id/submissions", h.ListAssignmentSubmissions)
		assignments.GET(":id/submissions/:submissionID", h.GetAssignmentSubmission)
		assignments.PATCH(":id/submissions/:submissionID/grade", h.GradeSubmission)

		submissions := api.Group("/assignments", studentGuard)
		submissions.POST(":id/submissions", h.SubmitAssignment)
		submissions.GET(":id", h.GetAssignment)
		submissions.GET(":id/submissions/me", h.GetMySubmission)

		notes := api.Group("/notes", studentGuard)
		notes.POST("", h.CreateNote)
		notes.GET("", h.ListMyNotes)
		notes.GET("/published", h.ListPublishedNotes)
		notes.PATCH(":id", h.UpdateNote)
		notes.DELETE(":id", h.DeleteNote)
		notes.POST(":id/restore", h.RestoreNote)
		notes.POST(":id/comments", h.CreateNoteComment)
		notes.GET(":id/comments", h.ListNoteComments)

		conversations := api.Group("/conversations", studentGuard)
		conversations.POST("", h.CreateConversation)
		conversations.GET("", h.ListConversations)
		conversations.GET(":id/messages", h.ListMessages)
		conversations.POST(":id/messages", h.SendMessage)
		conversations.POST(":id/read", h.MarkConversationRead)
	}
}

type loginRequest struct {
	SchoolID   string `json:"school_id" validate:"required"`
	Identifier string `json:"identifier" validate:"required"`
	Password   string `json:"password" validate:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	access, refresh, account, err := h.auth.Login(c.Request.Context(), req.SchoolID, req.Identifier, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAccountLocked):
			response.Error(c, http.StatusForbidden, "account locked", nil)
		case errors.Is(err, service.ErrPasswordResetRequired):
			response.Error(c, http.StatusForbidden, "password reset required", nil)
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Error(c, http.StatusUnauthorized, "invalid credentials", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "login failed", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"account": gin.H{
			"id":           account.ID,
			"school_id":    account.SchoolID,
			"role":         account.Role,
			"identifier":   account.Identifier,
			"display_name": account.DisplayName,
		},
	})
}

type createTeacherRequest struct {
	SchoolID   string `json:"school_id" validate:"required"`
	Number     string `json:"number" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Email      string `json:"email" validate:"required,email"`
	Phone      string `json:"phone" validate:"omitempty"`
	DefaultPwd string `json:"default_password" validate:"required"`
}

func (h *Handler) CreateTeacher(c *gin.Context) {
	var req createTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	teacher, err := h.admin.CreateTeacher(c.Request.Context(), service.CreateTeacherInput{
		SchoolID:   req.SchoolID,
		Number:     req.Number,
		Name:       req.Name,
		Email:      req.Email,
		Phone:      req.Phone,
		DefaultPwd: req.DefaultPwd,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to create teacher", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"teacher_id": teacher.ID})
}

type createStudentRequest struct {
	SchoolID   string   `json:"school_id" validate:"required"`
	Number     string   `json:"number" validate:"required"`
	Name       string   `json:"name" validate:"required"`
	Email      string   `json:"email" validate:"required,email"`
	Phone      string   `json:"phone"`
	ClassID    string   `json:"class_id" validate:"required"`
	TeacherIDs []string `json:"teacher_ids" validate:"required,min=1,dive,required"`
	DefaultPwd string   `json:"default_password" validate:"required"`
}

func (h *Handler) CreateStudent(c *gin.Context) {
	var req createStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	student, err := h.admin.CreateStudent(c.Request.Context(), service.CreateStudentInput{
		SchoolID:   req.SchoolID,
		Number:     req.Number,
		Name:       req.Name,
		Email:      req.Email,
		Phone:      req.Phone,
		ClassID:    req.ClassID,
		DefaultPwd: req.DefaultPwd,
		TeacherIDs: req.TeacherIDs,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to create student", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"student_id": student.ID})
}

func (h *Handler) ListAccounts(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	roleParam := strings.ToLower(strings.TrimSpace(c.DefaultQuery("role", "all")))
	var role domain.Role
	switch roleParam {
	case "", "all":
		role = ""
	case "teacher", "teachers":
		role = domain.RoleTeacher
	case "student", "students":
		role = domain.RoleStudent
	default:
		response.Error(c, http.StatusBadRequest, "invalid role", roleParam)
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page <= 0 {
		page = 1
	}
	size, err := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if err != nil || size <= 0 {
		size = 50
	}

	query := strings.TrimSpace(c.Query("query"))

	statusParam := strings.ToLower(strings.TrimSpace(c.Query("status")))
	var status domain.AccountStatus
	switch statusParam {
	case "", "all":
		status = ""
	case string(domain.AccountStatusActive):
		status = domain.AccountStatusActive
	case string(domain.AccountStatusLocked):
		status = domain.AccountStatusLocked
	case string(domain.AccountStatusPasswordResetRequired):
		status = domain.AccountStatusPasswordResetRequired
	default:
		response.Error(c, http.StatusBadRequest, "invalid status", statusParam)
		return
	}

	departmentID := strings.TrimSpace(c.Query("department_id"))
	classID := strings.TrimSpace(c.Query("class_id"))
	if classID != "" && departmentID == "" {
		response.Error(c, http.StatusBadRequest, "department_id required when class_id provided", nil)
		return
	}

	deptScopeParam := strings.ToLower(strings.TrimSpace(c.Query("department_scope")))
	var departmentScope service.AccountDepartmentScope
	switch deptScopeParam {
	case "", "all":
		departmentScope = service.AccountDepartmentScopeAll
	case string(service.AccountDepartmentScopeUnassigned):
		if departmentID != "" {
			response.Error(c, http.StatusBadRequest, "department_scope unassigned cannot be combined with department_id", nil)
			return
		}
		departmentScope = service.AccountDepartmentScopeUnassigned
	default:
		response.Error(c, http.StatusBadRequest, "invalid department_scope", deptScopeParam)
		return
	}

	classScopeParam := strings.ToLower(strings.TrimSpace(c.Query("class_scope")))
	var classScope service.AccountClassScope
	switch classScopeParam {
	case "", "all":
		classScope = service.AccountClassScopeAll
	case string(service.AccountClassScopeUnassigned):
		if classID != "" {
			response.Error(c, http.StatusBadRequest, "class_scope unassigned cannot be combined with class_id", nil)
			return
		}
		classScope = service.AccountClassScopeUnassigned
	default:
		response.Error(c, http.StatusBadRequest, "invalid class_scope", classScopeParam)
		return
	}

	accounts, total, err := h.admin.ListAccounts(c.Request.Context(), service.ListAccountsOptions{
		SchoolID:        schoolID,
		Role:            role,
		Status:          status,
		DepartmentID:    departmentID,
		DepartmentScope: departmentScope,
		ClassID:         classID,
		ClassScope:      classScope,
		Page:            page,
		Size:            size,
		Query:           query,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to list accounts", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"accounts":  accounts,
		"page":      page,
		"page_size": size,
		"total":     total,
	})
}

type accountActionRequest struct {
	SchoolID string `json:"school_id" validate:"required"`
}

type updateOssCredentialRequest struct {
	SchoolID             string  `json:"school_id" validate:"required"`
	Name                 *string `json:"name"`
	Endpoint             *string `json:"endpoint"`
	Region               *string `json:"region"`
	Bucket               *string `json:"bucket"`
	DirectoryPrefix      *string `json:"directory_prefix"`
	AccessKeyDisplay     *string `json:"access_key_display"`
	AllowPublicRead      *bool   `json:"allow_public_read"`
	AllowMultipartUpload *bool   `json:"allow_multipart_upload"`
	Active               *bool   `json:"active"`
	IsPrimary            *bool   `json:"is_primary"`
}

type updateOssPolicyRequest struct {
	SchoolID string `json:"school_id" validate:"required"`
	Status   string `json:"status" validate:"required"`
}

type createOssCredentialRequest struct {
	SchoolID             string `json:"school_id" validate:"required"`
	Name                 string `json:"name" validate:"required"`
	Endpoint             string `json:"endpoint" validate:"required"`
	Region               string `json:"region" validate:"required"`
	Bucket               string `json:"bucket" validate:"required"`
	DirectoryPrefix      string `json:"directory_prefix"`
	AccessKeyDisplay     string `json:"access_key_display"`
	AllowPublicRead      bool   `json:"allow_public_read"`
	AllowMultipartUpload bool   `json:"allow_multipart_upload"`
	Active               *bool  `json:"active"`
	IsPrimary            bool   `json:"is_primary"`
}

type createOssPolicyRequest struct {
	SchoolID    string `json:"school_id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	AppliesTo   string `json:"applies_to" validate:"required"`
	Status      string `json:"status"`
}

type updateSystemSwitchRequest struct {
	SchoolID string `json:"school_id" validate:"required"`
	Enabled  *bool  `json:"enabled" validate:"required"`
}

type updateSystemParameterRequest struct {
	SchoolID string `json:"school_id" validate:"required"`
	Value    string `json:"value" validate:"required"`
}

type updateSystemBroadcastRequest struct {
	SchoolID string `json:"school_id" validate:"required"`
	Status   string `json:"status"`
	Pinned   *bool  `json:"pinned"`
}

func (h *Handler) ResetAccountPassword(c *gin.Context) {
	accountID := strings.TrimSpace(c.Param("id"))
	if accountID == "" {
		response.Error(c, http.StatusBadRequest, "account id is required", nil)
		return
	}

	var req accountActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	if err := h.admin.ResetAccountPassword(c.Request.Context(), req.SchoolID, accountID); err != nil {
		h.handleAdminAccountError(c, err, "unable to reset password")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"account_id": accountID,
		"status":     domain.AccountStatusPasswordResetRequired,
	})
}

func (h *Handler) LockAccount(c *gin.Context) {
	accountID := strings.TrimSpace(c.Param("id"))
	if accountID == "" {
		response.Error(c, http.StatusBadRequest, "account id is required", nil)
		return
	}

	var req accountActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	if err := h.admin.LockAccount(c.Request.Context(), req.SchoolID, accountID); err != nil {
		h.handleAdminAccountError(c, err, "unable to lock account")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"account_id": accountID,
		"status":     domain.AccountStatusLocked,
	})
}

func (h *Handler) UnlockAccount(c *gin.Context) {
	accountID := strings.TrimSpace(c.Param("id"))
	if accountID == "" {
		response.Error(c, http.StatusBadRequest, "account id is required", nil)
		return
	}

	var req accountActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	if err := h.admin.UnlockAccount(c.Request.Context(), req.SchoolID, accountID); err != nil {
		h.handleAdminAccountError(c, err, "unable to unlock account")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"account_id": accountID,
		"status":     domain.AccountStatusActive,
	})
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	accountID := strings.TrimSpace(c.Param("id"))
	if accountID == "" {
		response.Error(c, http.StatusBadRequest, "account id is required", nil)
		return
	}

	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		var req accountActionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "invalid request payload", err.Error())
			return
		}
		if err := h.validate.Struct(req); err != nil {
			response.Error(c, http.StatusBadRequest, "validation error", err.Error())
			return
		}
		schoolID = req.SchoolID
	}
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	if err := h.admin.DeleteAccount(c.Request.Context(), schoolID, accountID); err != nil {
		h.handleAdminAccountError(c, err, "unable to delete account")
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"account_id": accountID,
	})
}

func (h *Handler) handleAdminAccountError(c *gin.Context, err error, message string) {
	switch {
	case errors.Is(err, service.ErrAdminAccountNotFound):
		response.Error(c, http.StatusNotFound, message, err.Error())
	case errors.Is(err, service.ErrAdminAccountRoleNotSupported):
		response.Error(c, http.StatusBadRequest, message, err.Error())
	case errors.Is(err, service.ErrAdminAccountAlreadyLocked):
		response.Error(c, http.StatusConflict, message, err.Error())
	case errors.Is(err, service.ErrAdminAccountNotLocked):
		response.Error(c, http.StatusConflict, message, err.Error())
	case errors.Is(err, service.ErrAdminPasswordResetPending):
		response.Error(c, http.StatusConflict, message, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, message, err.Error())
	}
}

type createDepartmentRequest struct {
	SchoolID string `json:"school_id" validate:"required"`
	Name     string `json:"name" validate:"required"`
}

func (h *Handler) CreateDepartment(c *gin.Context) {
	var req createDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	department, err := h.admin.CreateDepartment(c.Request.Context(), req.SchoolID, req.Name)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to create department", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"department_id": department.ID})
}

type updateDepartmentRequest struct {
	SchoolID string `json:"school_id" validate:"required"`
	Name     string `json:"name" validate:"required"`
}

func (h *Handler) UpdateDepartment(c *gin.Context) {
	departmentID := strings.TrimSpace(c.Param("id"))
	if departmentID == "" {
		response.Error(c, http.StatusBadRequest, "department id is required", nil)
		return
	}

	var req updateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	department, err := h.admin.UpdateDepartment(c.Request.Context(), req.SchoolID, departmentID, req.Name)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to update department", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"department": gin.H{
		"id":         department.ID,
		"school_id":  department.SchoolID,
		"name":       department.Name,
		"created_at": department.CreatedAt,
		"updated_at": department.UpdatedAt,
	}})
}

func (h *Handler) DeleteDepartment(c *gin.Context) {
	departmentID := strings.TrimSpace(c.Param("id"))
	if departmentID == "" {
		response.Error(c, http.StatusBadRequest, "department id is required", nil)
		return
	}
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id is required", nil)
		return
	}

	if err := h.admin.DeleteDepartment(c.Request.Context(), schoolID, departmentID); err != nil {
		response.Error(c, http.StatusBadRequest, "unable to delete department", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

type createClassRequest struct {
	SchoolID     string `json:"school_id" validate:"required"`
	DepartmentID string `json:"department_id" validate:"required"`
	Name         string `json:"name" validate:"required"`
}

func (h *Handler) CreateOssCredential(c *gin.Context) {
	var req createOssCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	operatorID := c.GetString(middleware.ContextAccountID)
	credential, err := h.oss.CreateCredential(c.Request.Context(), service.CreateOssCredentialInput{
		SchoolID:             req.SchoolID,
		Name:                 req.Name,
		Endpoint:             req.Endpoint,
		Region:               req.Region,
		Bucket:               req.Bucket,
		DirectoryPrefix:      req.DirectoryPrefix,
		AccessKeyDisplay:     req.AccessKeyDisplay,
		AllowPublicRead:      req.AllowPublicRead,
		AllowMultipartUpload: req.AllowMultipartUpload,
		Active:               active,
		IsPrimary:            req.IsPrimary,
		OperatorID:           operatorID,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to create credential", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"credential": credential})
}

func (h *Handler) DeleteOssCredential(c *gin.Context) {
	credentialID := strings.TrimSpace(c.Param("id"))
	if credentialID == "" {
		response.Error(c, http.StatusBadRequest, "credential id is required", nil)
		return
	}

	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		var req accountActionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "invalid request payload", err.Error())
			return
		}
		if err := h.validate.Struct(req); err != nil {
			response.Error(c, http.StatusBadRequest, "validation error", err.Error())
			return
		}
		schoolID = req.SchoolID
	}
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	operatorID := c.GetString(middleware.ContextAccountID)
	err := h.oss.DeleteCredential(c.Request.Context(), schoolID, credentialID, operatorID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOssPrimaryCredentialDeletion):
			response.Error(c, http.StatusBadRequest, "unable to delete credential", err.Error())
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Error(c, http.StatusNotFound, "credential not found", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "unable to delete credential", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) CreateOssPolicy(c *gin.Context) {
	var req createOssPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	statusValue := strings.ToLower(strings.TrimSpace(req.Status))
	var status domain.OssPolicyStatus
	switch statusValue {
	case "", string(domain.OssPolicyStatusEnabled):
		status = domain.OssPolicyStatusEnabled
	case string(domain.OssPolicyStatusReadOnly), "readonly":
		status = domain.OssPolicyStatusReadOnly
	case string(domain.OssPolicyStatusDisabled):
		status = domain.OssPolicyStatusDisabled
	default:
		response.Error(c, http.StatusBadRequest, "invalid status", statusValue)
		return
	}

	operatorID := c.GetString(middleware.ContextAccountID)
	policy, err := h.oss.CreatePolicy(c.Request.Context(), service.CreateOssPolicyInput{
		SchoolID:    req.SchoolID,
		Name:        req.Name,
		Description: req.Description,
		AppliesTo:   req.AppliesTo,
		Status:      status,
		OperatorID:  operatorID,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to create policy", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"policy": policy})
}

func (h *Handler) DeleteOssPolicy(c *gin.Context) {
	policyID := strings.TrimSpace(c.Param("id"))
	if policyID == "" {
		response.Error(c, http.StatusBadRequest, "policy id is required", nil)
		return
	}

	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		var req accountActionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "invalid request payload", err.Error())
			return
		}
		if err := h.validate.Struct(req); err != nil {
			response.Error(c, http.StatusBadRequest, "validation error", err.Error())
			return
		}
		schoolID = req.SchoolID
	}
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	operatorID := c.GetString(middleware.ContextAccountID)
	if err := h.oss.DeletePolicy(c.Request.Context(), schoolID, policyID, operatorID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "policy not found", err.Error())
		} else {
			response.Error(c, http.StatusInternalServerError, "unable to delete policy", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) ListOssCredentials(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	credentials, err := h.oss.ListCredentials(c.Request.Context(), schoolID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to list oss credentials", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"credentials": credentials})
}

func (h *Handler) UpdateOssCredential(c *gin.Context) {
	credentialID := strings.TrimSpace(c.Param("id"))
	if credentialID == "" {
		response.Error(c, http.StatusBadRequest, "credential id is required", nil)
		return
	}

	var req updateOssCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	operatorID := c.GetString(middleware.ContextAccountID)
	updated, err := h.oss.UpdateCredential(c.Request.Context(), service.UpdateOssCredentialInput{
		SchoolID:             req.SchoolID,
		CredentialID:         credentialID,
		Name:                 req.Name,
		Endpoint:             req.Endpoint,
		Region:               req.Region,
		Bucket:               req.Bucket,
		DirectoryPrefix:      req.DirectoryPrefix,
		AccessKeyDisplay:     req.AccessKeyDisplay,
		AllowPublicRead:      req.AllowPublicRead,
		AllowMultipartUpload: req.AllowMultipartUpload,
		Active:               req.Active,
		IsPrimary:            req.IsPrimary,
		OperatorID:           operatorID,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to update credential", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"credential": updated})
}

func (h *Handler) ListOssPolicies(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	policies, err := h.oss.ListPolicies(c.Request.Context(), schoolID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to list policies", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"policies": policies})
}

func (h *Handler) UpdateOssPolicy(c *gin.Context) {
	policyID := strings.TrimSpace(c.Param("id"))
	if policyID == "" {
		response.Error(c, http.StatusBadRequest, "policy id is required", nil)
		return
	}

	var req updateOssPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	statusValue := strings.ToLower(strings.TrimSpace(req.Status))
	var status domain.OssPolicyStatus
	switch statusValue {
	case string(domain.OssPolicyStatusEnabled):
		status = domain.OssPolicyStatusEnabled
	case string(domain.OssPolicyStatusReadOnly), "readonly":
		status = domain.OssPolicyStatusReadOnly
	case string(domain.OssPolicyStatusDisabled):
		status = domain.OssPolicyStatusDisabled
	default:
		response.Error(c, http.StatusBadRequest, "invalid status", statusValue)
		return
	}

	operatorID := c.GetString(middleware.ContextAccountID)
	policy, err := h.oss.UpdatePolicyStatus(c.Request.Context(), service.UpdateOssPolicyStatusInput{
		SchoolID:   req.SchoolID,
		PolicyID:   policyID,
		Status:     status,
		OperatorID: operatorID,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to update policy", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"policy": policy})
}

func (h *Handler) ListSystemSwitches(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}
	switches := h.system.ListSwitches(schoolID)
	response.Success(c, http.StatusOK, gin.H{"switches": switches})
}

func (h *Handler) UpdateSystemSwitch(c *gin.Context) {
	switchID := strings.TrimSpace(c.Param("id"))
	if switchID == "" {
		response.Error(c, http.StatusBadRequest, "switch id required", nil)
		return
	}

	var req updateSystemSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	updated, err := h.system.UpdateSwitchState(req.SchoolID, switchID, *req.Enabled, "")
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSystemSwitchNotFound):
			response.Error(c, http.StatusNotFound, "switch not found", nil)
		default:
			response.Error(c, http.StatusBadRequest, "failed to update switch", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"switch": updated})
}

func (h *Handler) ListSystemParameters(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}
	params := h.system.ListParameters(schoolID)
	response.Success(c, http.StatusOK, gin.H{"parameters": params})
}

func (h *Handler) UpdateSystemParameter(c *gin.Context) {
	parameterID := strings.TrimSpace(c.Param("id"))
	if parameterID == "" {
		response.Error(c, http.StatusBadRequest, "parameter id required", nil)
		return
	}

	var req updateSystemParameterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	updated, err := h.system.UpdateParameter(req.SchoolID, parameterID, req.Value, "")
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSystemParameterNotFound):
			response.Error(c, http.StatusNotFound, "parameter not found", nil)
		default:
			response.Error(c, http.StatusBadRequest, "failed to update parameter", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"parameter": updated})
}

func (h *Handler) ListSystemBroadcasts(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}
	items := h.system.ListBroadcasts(schoolID)
	response.Success(c, http.StatusOK, gin.H{"broadcasts": items})
}

func (h *Handler) UpdateSystemBroadcast(c *gin.Context) {
	broadcastID := strings.TrimSpace(c.Param("id"))
	if broadcastID == "" {
		response.Error(c, http.StatusBadRequest, "broadcast id required", nil)
		return
	}

	var req updateSystemBroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	var statusPtr *service.AdminSystemBroadcastStatus
	statusValue := strings.ToLower(strings.TrimSpace(req.Status))
	if statusValue != "" {
		switch statusValue {
		case string(service.AdminSystemBroadcastScheduled):
			v := service.AdminSystemBroadcastScheduled
			statusPtr = &v
		case string(service.AdminSystemBroadcastSent):
			v := service.AdminSystemBroadcastSent
			statusPtr = &v
		case string(service.AdminSystemBroadcastDraft):
			v := service.AdminSystemBroadcastDraft
			statusPtr = &v
		default:
			response.Error(c, http.StatusBadRequest, "invalid status", statusValue)
			return
		}
	}

	if statusPtr == nil && req.Pinned == nil {
		response.Error(c, http.StatusBadRequest, "status or pinned must be provided", nil)
		return
	}

	updated, err := h.system.UpdateBroadcast(req.SchoolID, broadcastID, statusPtr, req.Pinned, "")
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSystemBroadcastNotFound):
			response.Error(c, http.StatusNotFound, "broadcast not found", nil)
		default:
			response.Error(c, http.StatusBadRequest, "failed to update broadcast", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"broadcast": updated})
}

func (h *Handler) ListSystemAuditLogs(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}
	limitParam := strings.TrimSpace(c.DefaultQuery("limit", "0"))
	limit, err := strconv.Atoi(limitParam)
	if err != nil {
		limit = 0
	}
	logs := h.system.ListAuditLogs(schoolID, limit)
	response.Success(c, http.StatusOK, gin.H{"logs": logs})
}
func (h *Handler) ListOssAuditLogs(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	limitParam := strings.TrimSpace(c.DefaultQuery("limit", "20"))
	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 {
		limit = 20
	}

	logs, err := h.oss.ListAuditLogs(c.Request.Context(), schoolID, limit)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to list audit logs", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"logs": logs})
}

func (h *Handler) CreateClass(c *gin.Context) {
	var req createClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	class, err := h.admin.CreateClass(c.Request.Context(), req.SchoolID, req.DepartmentID, req.Name)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to create class", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"class_id": class.ID})
}

type updateClassRequest struct {
	SchoolID string `json:"school_id" validate:"required"`
	Name     string `json:"name" validate:"required"`
}

func (h *Handler) UpdateClass(c *gin.Context) {
	classID := strings.TrimSpace(c.Param("id"))
	if classID == "" {
		response.Error(c, http.StatusBadRequest, "class id is required", nil)
		return
	}

	var req updateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	class, err := h.admin.UpdateClass(c.Request.Context(), req.SchoolID, classID, req.Name)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to update class", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"class": gin.H{
		"id":            class.ID,
		"school_id":     class.SchoolID,
		"department_id": class.DepartmentID,
		"name":          class.Name,
		"created_at":    class.CreatedAt,
		"updated_at":    class.UpdatedAt,
	}})
}

func (h *Handler) DeleteClass(c *gin.Context) {
	classID := strings.TrimSpace(c.Param("id"))
	if classID == "" {
		response.Error(c, http.StatusBadRequest, "class id is required", nil)
		return
	}
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id is required", nil)
		return
	}

	if err := h.admin.DeleteClass(c.Request.Context(), schoolID, classID); err != nil {
		response.Error(c, http.StatusBadRequest, "unable to delete class", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) ListDepartments(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id is required", nil)
		return
	}

	departments, err := h.admin.ListDepartments(c.Request.Context(), schoolID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list departments", err.Error())
		return
	}

	items := make([]gin.H, 0, len(departments))
	for _, dept := range departments {
		items = append(items, gin.H{
			"id":         dept.ID,
			"school_id":  dept.SchoolID,
			"name":       dept.Name,
			"created_at": dept.CreatedAt,
			"updated_at": dept.UpdatedAt,
		})
	}

	response.Success(c, http.StatusOK, gin.H{"departments": items})
}

func (h *Handler) ListClasses(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id is required", nil)
		return
	}

	departmentID := strings.TrimSpace(c.Param("id"))
	if departmentID == "" {
		response.Error(c, http.StatusBadRequest, "department id is required", nil)
		return
	}

	classes, err := h.admin.ListClasses(c.Request.Context(), schoolID, departmentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list classes", err.Error())
		return
	}

	items := make([]gin.H, 0, len(classes))
	for _, class := range classes {
		payload := gin.H{
			"id":            class.ID,
			"school_id":     class.SchoolID,
			"department_id": class.DepartmentID,
			"name":          class.Name,
			"created_at":    class.CreatedAt,
			"updated_at":    class.UpdatedAt,
		}
		if class.HomeroomID != nil {
			payload["homeroom_id"] = class.HomeroomID
		}
		items = append(items, payload)
	}

	response.Success(c, http.StatusOK, gin.H{"classes": items})
}

type createAssignmentRequest struct {
	CourseID      string                          `json:"course_id" validate:"required"`
	TeacherID     string                          `json:"teacher_id" validate:"required"`
	ClassID       string                          `json:"class_id" validate:"required"`
	Type          string                          `json:"type" validate:"required,oneof=homework exam"`
	Title         string                          `json:"title" validate:"required"`
	Description   string                          `json:"description"`
	StartAt       *service.TimeISO8601            `json:"start_at"`
	DueAt         *service.TimeISO8601            `json:"due_at"`
	MaxScore      float64                         `json:"max_score" validate:"gte=0"`
	AllowResubmit bool                            `json:"allow_resubmit"`
	Questions     []createAssignmentQuestionInput `json:"questions" validate:"required,min=1,dive"`
}

type createAssignmentQuestionInput struct {
	Type       string  `json:"type" validate:"required,oneof=fill choice judge essay"`
	Prompt     string  `json:"prompt" validate:"required"`
	Options    string  `json:"options"`
	Answer     string  `json:"answer"`
	Score      float64 `json:"score" validate:"gte=0"`
	OrderIndex int     `json:"order_index"`
}

func (h *Handler) CreateAssignment(c *gin.Context) {
	var req createAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	startAt := convertToTime(req.StartAt)
	dueAt := convertToTime(req.DueAt)

	questions := make([]service.QuestionInput, 0, len(req.Questions))
	for _, q := range req.Questions {
		questions = append(questions, service.QuestionInput{
			Type:       service.ToQuestionType(q.Type),
			Prompt:     q.Prompt,
			Options:    q.Options,
			Answer:     q.Answer,
			Score:      q.Score,
			OrderIndex: q.OrderIndex,
		})
	}

	assignment, err := h.assignments.CreateAssignment(c.Request.Context(), service.CreateAssignmentInput{
		CourseID:      req.CourseID,
		TeacherID:     req.TeacherID,
		ClassID:       req.ClassID,
		Type:          service.ToAssignmentType(req.Type),
		Title:         req.Title,
		Description:   req.Description,
		StartAt:       startAt,
		DueAt:         dueAt,
		MaxScore:      req.MaxScore,
		AllowResubmit: req.AllowResubmit,
		Questions:     questions,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to create assignment", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"assignment_id": assignment.ID})
}

func (h *Handler) GetAssignment(c *gin.Context) {
	assignmentID := strings.TrimSpace(c.Param("id"))
	if assignmentID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment id", nil)
		return
	}

	assignment, questions, err := h.assignments.GetAssignment(c.Request.Context(), assignmentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssignmentNotFound):
			response.Error(c, http.StatusNotFound, "assignment not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to load assignment", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"assignment": assignmentPayload(*assignment, questions)})
}

func (h *Handler) ListAssignmentSubmissions(c *gin.Context) {
	assignmentID := strings.TrimSpace(c.Param("id"))
	if assignmentID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment id", nil)
		return
	}

	details, err := h.assignments.ListAssignmentSubmissions(c.Request.Context(), assignmentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list submissions", err.Error())
		return
	}

	payload := make([]gin.H, 0, len(details))
	for _, detail := range details {
		payload = append(payload, submissionDetailPayload(detail))
	}

	response.Success(c, http.StatusOK, gin.H{"submissions": payload})
}

type submitAssignmentRequest struct {
	StudentID string                   `json:"student_id" validate:"required"`
	Status    string                   `json:"status" validate:"required"`
	Score     *float64                 `json:"score"`
	Feedback  string                   `json:"feedback"`
	Answers   []submitAssignmentAnswer `json:"answers" validate:"required,min=1,dive"`
}

type submitAssignmentAnswer struct {
	QuestionID string   `json:"question_id" validate:"required"`
	Answer     string   `json:"answer" validate:"required"`
	Score      *float64 `json:"score"`
}

type submissionCommentRequest struct {
	Content string `json:"content"`
}

type gradeSubmissionRequest struct {
	Score      *float64                  `json:"score"`
	Feedback   string                    `json:"feedback"`
	ItemScores map[string]*float64       `json:"item_scores"`
	Comment    *submissionCommentRequest `json:"comment"`
}

type createNoteRequest struct {
	Title      string `json:"title" validate:"required"`
	Content    string `json:"content" validate:"required"`
	Visibility string `json:"visibility" validate:"required,oneof=private class school"`
	Status     string `json:"status" validate:"required,oneof=draft published"`
}

type updateNoteRequest struct {
	Title      *string `json:"title"`
	Content    *string `json:"content"`
	Visibility *string `json:"visibility"`
	Status     *string `json:"status"`
}

type createNoteCommentRequest struct {
	Content string `json:"content" validate:"required"`
}

type createConversationRequest struct {
	ParticipantIDs []string `json:"participant_ids" validate:"required,min=1,dive,required"`
}

type sendMessageRequest struct {
	Kind     string `json:"kind" validate:"required,oneof=text image video audio file"`
	Text     string `json:"text"`
	MediaURI string `json:"media_uri"`
	Metadata string `json:"metadata"`
}

type markConversationReadRequest struct {
	MessageID string `json:"message_id" validate:"required"`
}

func (h *Handler) SubmitAssignment(c *gin.Context) {
	assignmentID := c.Param("id")
	if assignmentID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment id", nil)
		return
	}

	var req submitAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	answers := make([]service.AnswerInput, 0, len(req.Answers))
	for _, ans := range req.Answers {
		answers = append(answers, service.AnswerInput{
			QuestionID: ans.QuestionID,
			Answer:     ans.Answer,
			Score:      ans.Score,
		})
	}

	err := h.assignments.Submit(c.Request.Context(), service.SubmitAssignmentInput{
		AssignmentID: assignmentID,
		StudentID:    req.StudentID,
		Answers:      answers,
		Score:        req.Score,
		Feedback:     req.Feedback,
		Status:       req.Status,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to submit assignment", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"submission": "ok"})
}

func (h *Handler) GetMySubmission(c *gin.Context) {
	assignmentID := strings.TrimSpace(c.Param("id"))
	if assignmentID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment id", nil)
		return
	}

	studentID := getAccountID(c)
	if studentID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	detail, comments, err := h.assignments.GetSubmissionForStudent(c.Request.Context(), assignmentID, studentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssignmentNotFound):
			response.Error(c, http.StatusNotFound, "assignment not found", nil)
		case errors.Is(err, service.ErrSubmissionNotFound):
			response.Error(c, http.StatusNotFound, "submission not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to load submission", err.Error())
		}
		return
	}

	payload := gin.H{
		"submission": submissionDetailPayload(*detail),
		"comments":   submissionCommentsPayload(comments),
	}
	response.Success(c, http.StatusOK, payload)
}

func (h *Handler) GetAssignmentSubmission(c *gin.Context) {
	assignmentID := strings.TrimSpace(c.Param("id"))
	submissionID := strings.TrimSpace(c.Param("submissionID"))
	if assignmentID == "" || submissionID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment or submission id", nil)
		return
	}

	teacherID := getAccountID(c)
	if teacherID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	detail, comments, err := h.assignments.GetSubmissionForTeacher(c.Request.Context(), teacherID, assignmentID, submissionID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssignmentNotFound):
			response.Error(c, http.StatusNotFound, "assignment not found", nil)
		case errors.Is(err, service.ErrSubmissionNotFound):
			response.Error(c, http.StatusNotFound, "submission not found", nil)
		case errors.Is(err, service.ErrSubmissionForbidden):
			response.Error(c, http.StatusForbidden, "submission forbidden", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to load submission", err.Error())
		}
		return
	}

	payload := gin.H{
		"submission": submissionDetailPayload(*detail),
		"comments":   submissionCommentsPayload(comments),
	}
	response.Success(c, http.StatusOK, payload)
}

func (h *Handler) GradeSubmission(c *gin.Context) {
	assignmentID := strings.TrimSpace(c.Param("id"))
	submissionID := strings.TrimSpace(c.Param("submissionID"))
	if assignmentID == "" || submissionID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment or submission id", nil)
		return
	}

	var req gradeSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if req.Comment != nil && strings.TrimSpace(req.Comment.Content) == "" {
		response.Error(c, http.StatusBadRequest, "comment content cannot be empty", nil)
		return
	}

	teacherID := getAccountID(c)
	if teacherID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	input := service.GradeSubmissionInput{
		AssignmentID: assignmentID,
		SubmissionID: submissionID,
		Score:        req.Score,
		Feedback:     req.Feedback,
		ItemScores:   req.ItemScores,
	}
	if req.Comment != nil {
		input.Comment = &service.SubmissionCommentInput{Content: req.Comment.Content}
	}

	detail, comments, err := h.assignments.GradeSubmission(c.Request.Context(), teacherID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssignmentNotFound):
			response.Error(c, http.StatusNotFound, "assignment not found", nil)
		case errors.Is(err, service.ErrSubmissionNotFound):
			response.Error(c, http.StatusNotFound, "submission not found", nil)
		case errors.Is(err, service.ErrSubmissionForbidden):
			response.Error(c, http.StatusForbidden, "submission forbidden", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to grade submission", err.Error())
		}
		return
	}

	payload := gin.H{
		"submission": submissionDetailPayload(*detail),
		"comments":   submissionCommentsPayload(comments),
	}
	response.Success(c, http.StatusOK, payload)
}

func (h *Handler) CreateNote(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	var req createNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	note, err := h.notes.CreateNote(c.Request.Context(), accountID, service.CreateNoteInput{
		Title:      req.Title,
		Content:    req.Content,
		Visibility: req.Visibility,
		Status:     req.Status,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to create note", err.Error())
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"note": notePayload(*note)})
}

func (h *Handler) UpdateNote(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		response.Error(c, http.StatusBadRequest, "missing note id", nil)
		return
	}

	var req updateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	if req.Title == nil && req.Content == nil && req.Visibility == nil && req.Status == nil {
		response.Error(c, http.StatusBadRequest, "no fields to update", nil)
		return
	}

	input := service.UpdateNoteInput{}
	if req.Title != nil {
		input.Title = *req.Title
	}
	if req.Content != nil {
		input.Content = *req.Content
	}
	if req.Visibility != nil {
		input.Visibility = *req.Visibility
	}
	if req.Status != nil {
		input.Status = *req.Status
	}

	note, err := h.notes.UpdateNote(c.Request.Context(), accountID, noteID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoteNotFound):
			response.Error(c, http.StatusNotFound, "note not found", nil)
		case errors.Is(err, service.ErrNoteForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to access note", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to update note", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"note": notePayload(*note)})
}

func (h *Handler) ListMyNotes(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	includeDeleted := false
	if raw := c.Query("include_deleted"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid include_deleted query", err.Error())
			return
		}
		includeDeleted = parsed
	}

	notes, err := h.notes.ListMyNotes(c.Request.Context(), accountID, c.Query("status"), includeDeleted)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list notes", err.Error())
		return
	}

	payload := make([]gin.H, 0, len(notes))
	for _, note := range notes {
		payload = append(payload, notePayload(note))
	}

	response.Success(c, http.StatusOK, gin.H{"notes": payload})
}

func (h *Handler) ListPublishedNotes(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	notes, err := h.notes.ListPublishedNotes(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list published notes", err.Error())
		return
	}

	payload := make([]gin.H, 0, len(notes))
	for _, note := range notes {
		payload = append(payload, notePayload(note))
	}

	response.Success(c, http.StatusOK, gin.H{"notes": payload})
}

func (h *Handler) DeleteNote(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		response.Error(c, http.StatusBadRequest, "missing note id", nil)
		return
	}

	if err := h.notes.DeleteNote(c.Request.Context(), accountID, noteID); err != nil {
		switch {
		case errors.Is(err, service.ErrNoteNotFound):
			response.Error(c, http.StatusNotFound, "note not found", nil)
		case errors.Is(err, service.ErrNoteForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to access note", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to delete note", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"deleted": true})
}

func (h *Handler) RestoreNote(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		response.Error(c, http.StatusBadRequest, "missing note id", nil)
		return
	}

	if err := h.notes.RestoreNote(c.Request.Context(), accountID, noteID); err != nil {
		switch {
		case errors.Is(err, service.ErrNoteNotFound):
			response.Error(c, http.StatusNotFound, "note not found", nil)
		case errors.Is(err, service.ErrNoteForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to access note", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to restore note", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"restored": true})
}

func (h *Handler) CreateNoteComment(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		response.Error(c, http.StatusBadRequest, "missing note id", nil)
		return
	}

	var req createNoteCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	comment, err := h.noteComments.AddComment(c.Request.Context(), accountID, noteID, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoteNotFound):
			response.Error(c, http.StatusNotFound, "note not found", nil)
		case errors.Is(err, service.ErrNoteCommentNotAllowed):
			response.Error(c, http.StatusForbidden, "not allowed to comment", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to create comment", err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"comment": noteCommentPayload(*comment)})
}

func (h *Handler) ListNoteComments(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		response.Error(c, http.StatusBadRequest, "missing note id", nil)
		return
	}

	comments, err := h.noteComments.ListComments(c.Request.Context(), accountID, noteID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNoteNotFound):
			response.Error(c, http.StatusNotFound, "note not found", nil)
		case errors.Is(err, service.ErrNoteCommentNotAllowed):
			response.Error(c, http.StatusForbidden, "not allowed to view comments", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to list comments", err.Error())
		}
		return
	}

	payload := make([]gin.H, 0, len(comments))
	for _, comment := range comments {
		payload = append(payload, noteCommentPayload(comment))
	}

	response.Success(c, http.StatusOK, gin.H{"comments": payload})
}

func (h *Handler) CreateConversation(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	if len(req.ParticipantIDs) != 1 {
		response.Error(c, http.StatusBadRequest, "only one participant supported for direct conversation", nil)
		return
	}

	summary, err := h.conversations.CreateDirectConversation(c.Request.Context(), accountID, req.ParticipantIDs[0])
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationInvalid):
			response.Error(c, http.StatusBadRequest, "invalid conversation", err.Error())
		case errors.Is(err, service.ErrConversationForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to create conversation", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to create conversation", err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"conversation": conversationPayload(*summary)})
}

func (h *Handler) ListConversations(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	convs, err := h.conversations.ListConversations(c.Request.Context(), accountID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list conversations", err.Error())
		return
	}

	payload := make([]gin.H, 0, len(convs))
	for _, conv := range convs {
		payload = append(payload, conversationPayload(conv))
	}

	response.Success(c, http.StatusOK, gin.H{"conversations": payload})
}

func (h *Handler) SendMessage(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		response.Error(c, http.StatusBadRequest, "missing conversation id", nil)
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	if req.Kind == "text" {
		if strings.TrimSpace(req.Text) == "" {
			response.Error(c, http.StatusBadRequest, "text message requires non-empty text", nil)
			return
		}
	} else if req.MediaURI == "" {
		response.Error(c, http.StatusBadRequest, "media message requires media_uri", nil)
		return
	}

	msg, err := h.conversations.SendMessage(c.Request.Context(), accountID, service.SendMessageInput{
		ConversationID: conversationID,
		Kind:           req.Kind,
		Text:           req.Text,
		MediaURI:       req.MediaURI,
		Metadata:       req.Metadata,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to send message", nil)
		case errors.Is(err, service.ErrConversationNotFound):
			response.Error(c, http.StatusNotFound, "conversation not found", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to send message", err.Error())
		}
		return
	}

	payload := messagePayload(*msg)
	response.Success(c, http.StatusCreated, gin.H{"message": payload})

	protoMsg := grpcmapper.Message(*msg)
	h.streamHub.Broadcast(conversationID, &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_MessageEvent{
			MessageEvent: &grpcpb.MessageCreatedEvent{Message: protoMsg},
		},
	})
}

func (h *Handler) ListMessages(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		response.Error(c, http.StatusBadRequest, "missing conversation id", nil)
		return
	}

	limit := 50
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			response.Error(c, http.StatusBadRequest, "invalid limit", raw)
			return
		}
		limit = parsed
	}
	beforeID := c.Query("before_id")

	messages, err := h.conversations.ListMessages(c.Request.Context(), accountID, conversationID, limit, beforeID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to view messages", nil)
		case errors.Is(err, service.ErrConversationNotFound):
			response.Error(c, http.StatusNotFound, "conversation not found", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to list messages", err.Error())
		}
		return
	}

	payload := make([]gin.H, 0, len(messages))
	for _, msg := range messages {
		payload = append(payload, messagePayload(msg))
	}

	response.Success(c, http.StatusOK, gin.H{"messages": payload})
}

func (h *Handler) MarkConversationRead(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	conversationID := c.Param("id")
	if conversationID == "" {
		response.Error(c, http.StatusBadRequest, "missing conversation id", nil)
		return
	}

	var req markConversationReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	if err := h.conversations.MarkRead(c.Request.Context(), accountID, conversationID, req.MessageID); err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to mark read", nil)
		case errors.Is(err, service.ErrConversationNotFound):
			response.Error(c, http.StatusNotFound, "conversation or message not found", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to mark read", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"read": true})

	h.streamHub.Broadcast(conversationID, &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_ReadEvent{
			ReadEvent: &grpcpb.ConversationReadEvent{
				MessageId: req.MessageID,
				ReaderId:  accountID,
			},
		},
	})
}

func assignmentPayload(assignment domain.Assignment, questions []domain.AssignmentQuestion) gin.H {
	questionsPayload := make([]gin.H, 0, len(questions))
	for _, q := range questions {
		questionsPayload = append(questionsPayload, assignmentQuestionPayload(q))
	}

	return gin.H{
		"id":             assignment.ID,
		"course_id":      assignment.CourseID,
		"teacher_id":     assignment.TeacherID,
		"class_id":       assignment.ClassID,
		"type":           assignment.Type,
		"title":          assignment.Title,
		"description":    assignment.Description,
		"start_at":       assignment.StartAt,
		"due_at":         assignment.DueAt,
		"max_score":      assignment.MaxScore,
		"allow_resubmit": assignment.AllowResubmit,
		"created_at":     assignment.CreatedAt,
		"updated_at":     assignment.UpdatedAt,
		"questions":      questionsPayload,
	}
}

func assignmentQuestionPayload(q domain.AssignmentQuestion) gin.H {
	return gin.H{
		"id":            q.ID,
		"assignment_id": q.AssignmentID,
		"type":          q.Type,
		"prompt":        q.Prompt,
		"options":       q.Options,
		"answer":        q.Answer,
		"score":         q.Score,
		"order_index":   q.OrderIndex,
	}
}

func submissionDetailPayload(detail service.SubmissionDetail) gin.H {
	items := make([]gin.H, 0, len(detail.Items))
	for _, item := range detail.Items {
		items = append(items, gin.H{
			"id":            item.ID,
			"submission_id": item.SubmissionID,
			"question_id":   item.QuestionID,
			"answer":        item.Answer,
			"score":         item.Score,
		})
	}

	return gin.H{
		"id":            detail.Submission.ID,
		"assignment_id": detail.Submission.AssignmentID,
		"student_id":    detail.Submission.StudentID,
		"status":        detail.Submission.Status,
		"score":         detail.Submission.Score,
		"feedback":      detail.Submission.Feedback,
		"submitted_at":  detail.Submission.SubmittedAt,
		"created_at":    detail.Submission.CreatedAt,
		"updated_at":    detail.Submission.UpdatedAt,
		"items":         items,
	}
}

func submissionCommentPayload(comment domain.SubmissionComment) gin.H {
	return gin.H{
		"id":             comment.ID,
		"submission_id":  comment.SubmissionID,
		"author_id":      comment.AuthorID,
		"author_role":    string(comment.AuthorRole),
		"content":        comment.Content,
		"attachment_uri": comment.AttachmentURI,
		"created_at":     comment.CreatedAt,
	}
}

func submissionCommentsPayload(comments []domain.SubmissionComment) []gin.H {
	payload := make([]gin.H, 0, len(comments))
	for _, comment := range comments {
		payload = append(payload, submissionCommentPayload(comment))
	}
	return payload
}

func conversationPayload(summary service.ConversationSummary) gin.H {
	members := make([]gin.H, 0, len(summary.Members))
	for _, member := range summary.Members {
		members = append(members, gin.H{
			"id":              member.ID,
			"conversation_id": member.ConversationID,
			"account_id":      member.AccountID,
			"role":            string(member.Role),
			"created_at":      member.CreatedAt,
		})
	}

	var last interface{}
	if summary.LastMessage != nil {
		last = messagePayload(*summary.LastMessage)
	}

	return gin.H{
		"id":           summary.Conversation.ID,
		"type":         summary.Conversation.Type,
		"school_id":    summary.Conversation.SchoolID,
		"created_at":   summary.Conversation.CreatedAt,
		"updated_at":   summary.Conversation.UpdatedAt,
		"members":      members,
		"last_message": last,
		"unread_count": summary.UnreadCount,
	}
}

func messagePayload(msg domain.Message) gin.H {
	return gin.H{
		"id":              msg.ID,
		"conversation_id": msg.ConversationID,
		"sender_id":       msg.SenderID,
		"sender_role":     string(msg.SenderRole),
		"kind":            msg.Kind,
		"text":            msg.Text,
		"media_uri":       msg.MediaURI,
		"metadata":        msg.Metadata,
		"created_at":      msg.CreatedAt,
	}
}

func noteCommentPayload(comment domain.NoteComment) gin.H {
	return gin.H{
		"id":          comment.ID,
		"note_id":     comment.NoteID,
		"author_id":   comment.AuthorID,
		"author_role": string(comment.AuthorRole),
		"content":     comment.Content,
		"created_at":  comment.CreatedAt,
	}
}

func notePayload(note domain.Note) gin.H {
	var deletedAt interface{}
	if note.DeletedAt != nil {
		deletedAt = note.DeletedAt
	}

	return gin.H{
		"id":         note.ID,
		"school_id":  note.SchoolID,
		"owner_id":   note.OwnerID,
		"owner_role": string(note.OwnerRole),
		"title":      note.Title,
		"content":    note.Content,
		"visibility": note.Visibility,
		"status":     note.Status,
		"deleted_at": deletedAt,
		"created_at": note.CreatedAt,
		"updated_at": note.UpdatedAt,
	}
}

func getAccountID(c *gin.Context) string {
	return c.GetString(middleware.ContextAccountID)
}

func convertToTime(t *service.TimeISO8601) *time.Time {
	if t == nil {
		return nil
	}
	parsed := t.Time
	return &parsed
}
