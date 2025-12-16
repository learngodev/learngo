package http

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

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
	teacher       *service.TeacherPortalService
	student       *service.StudentPortalService
	notes         *service.NoteService
	noteComments  *service.NoteCommentService
	conversations *service.ConversationService
	oss           *service.AdminOssService
	system        *service.AdminSystemService
	ai            *service.AIAssistantService
	aiGrading     *service.AIGradingService
	school        *service.SchoolService
	courseService *service.CourseService
	schedule      *service.ScheduleService
	streamHub     *realtime.Hub
	validate      *validator.Validate
}

// NewHandler constructs a Handler instance.
func NewHandler(auth *service.AuthService, admin *service.AdminService, assignments *service.AssignmentService, teacher *service.TeacherPortalService, student *service.StudentPortalService, conversations *service.ConversationService, notes *service.NoteService, noteComments *service.NoteCommentService, oss *service.AdminOssService, system *service.AdminSystemService, ai *service.AIAssistantService, aiGrading *service.AIGradingService, school *service.SchoolService, courseService *service.CourseService, schedule *service.ScheduleService, streamHub *realtime.Hub) *Handler {
	return &Handler{
		auth:          auth,
		admin:         admin,
		assignments:   assignments,
		teacher:       teacher,
		student:       student,
		notes:         notes,
		noteComments:  noteComments,
		conversations: conversations,
		oss:           oss,
		system:        system,
		ai:            ai,
		aiGrading:     aiGrading,
		school:        school,
		courseService: courseService,
		schedule:      schedule,
		streamHub:     streamHub,
		validate:      validator.New(),
	}
}

// RegisterRoutes attaches HTTP endpoints to router.
func (h *Handler) RegisterRoutes(r *gin.Engine, adminGuard gin.HandlerFunc, teacherGuard gin.HandlerFunc, studentGuard gin.HandlerFunc) {
	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", h.Login)
		api.POST("/auth/refresh", h.RefreshToken)
		api.POST("/auth/password/reset/request", h.RequestPasswordReset)
		api.POST("/auth/password/reset/confirm", h.ConfirmPasswordReset)
		api.GET("/schools", h.ListSchools)

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
		admin.POST("/accounts/batch", h.BatchOperateAccounts)
		admin.PATCH("/accounts/:id/structure", h.UpdateAccountStructure)
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
		admin.GET("/ai/settings", h.GetAIAgentSetting)
		admin.PUT("/ai/settings", h.UpdateAIAgentSetting)
		admin.GET("/ai/settings/audit_logs", h.ListAIAgentSettingAudits)
		admin.GET("/ai/usage", h.ListAIUsageSummaries)
		admin.GET("/ai/usage/export", h.ExportAIUsageSummaries)
		admin.GET("/ai/usage/report", h.GetAIUsageReport)
		admin.GET("/ai/usage/timeline", h.GetAIUsageTimeline)
		admin.GET("/time-slots", h.ListTimeSlots)
		admin.POST("/time-slots", h.CreateTimeSlot)
		admin.PATCH("/time-slots/:id", h.UpdateTimeSlot)
		admin.DELETE("/time-slots/:id", h.DeleteTimeSlot)

		admin.POST("/schedules/rules", h.CreateSchedule)
		admin.GET("/schedules/rules", h.ListSchedules)
		admin.GET("/schedules/stats", h.GetScheduleStats)
		admin.POST("/schedules/generate", h.GenerateSessions)
		admin.GET("/schedules/slots", h.ListTimeSlots)
		admin.POST("/schedules/slots", h.CreateTimeSlot)

		admin.GET("/courses", h.ListCourses)
		admin.POST("/courses", h.CreateCourse)
		admin.PATCH("/courses/:id", h.UpdateCourse)
		admin.DELETE("/courses/:id", h.DeleteCourse)
		admin.POST("/courses/assign", h.AssignCourse)
		admin.POST("/courses/assign/batch", h.BatchAssignCourse)
		admin.GET("/courses/assignments", h.ListAssignments)
		admin.DELETE("/courses/assignments/:id", h.RemoveAssignment)
		admin.POST("/courses/assignments/batch-remove", h.BatchRemoveAssignments)
		admin.POST("/courses/:id/assign/students", h.AssignStudents)
		admin.POST("/courses/:id/assign/teachers", h.AssignTeachers)

		assignments := api.Group("/assignments", teacherGuard)
		assignments.POST("", h.CreateAssignment)
		assignments.PATCH(":id", h.UpdateAssignment)
		assignments.GET(":id/submissions", h.ListAssignmentSubmissions)
		assignments.GET(":id/submissions/:submissionID", h.GetAssignmentSubmission)
		assignments.PATCH(":id/submissions/:submissionID/grade", h.GradeSubmission)
		assignments.POST(":id/submissions/:submissionID/return", h.ReturnSubmission)

		submissions := api.Group("/assignments", studentGuard)
		submissions.POST(":id/submissions", h.SubmitAssignment)
		submissions.GET(":id", h.GetAssignment)
		submissions.GET(":id/submissions/me", h.GetMySubmission)

		student := api.Group("/student", studentGuard)
		student.GET("/assignments", h.ListStudentAssignments)
		student.GET("/schedule", h.ListStudentSchedule)
		student.GET("/time-slots", h.ListTimeSlots)
		student.GET("/agenda", h.ListStudentAgenda)
		student.GET("/exams", h.ListStudentExams)
		student.GET("/reminders", h.ListStudentReminders)
		student.POST("/reminders", h.CreateStudentReminder)
		student.PATCH("/reminders/:id", h.UpdateStudentReminder)
		student.POST("/reminders/:id/completion", h.UpdateStudentReminderCompletion)
		student.POST("/reminders/completion/batch", h.BatchUpdateStudentReminderCompletion)
		student.POST("/reminders/completion/all", h.UpdateAllStudentRemindersCompletion)
		student.DELETE("/reminders/:id", h.DeleteStudentReminder)
		student.POST("/reminders/complete_all", h.UpdateAllStudentRemindersCompletion)

		teacher := api.Group("/teacher", teacherGuard)
		teacher.GET("/schedule", h.ListTeacherSchedule)
		teacher.GET("/time-slots", h.ListTimeSlots)
		teacher.GET("/assignments", h.ListTeacherAssignments)
		teacher.GET("/exams", h.ListTeacherExams)
		teacher.GET("/assignments/:id", h.GetTeacherAssignment)
		teacher.GET("/assignments/:id/export", h.ExportTeacherAssignmentGrades)
		teacher.GET("/agenda", h.ListTeacherAgenda)
		teacher.POST("/grade_assignment", h.GradeAssignment)
		teacher.POST("/generate_questions", h.GenerateQuestions)
		teacher.PATCH("/sessions/:id", h.UpdateSession)

		notes := api.Group("/notes", studentGuard)
		notes.POST("", h.CreateNote)
		notes.GET("", h.ListMyNotes)
		notes.GET("/published", h.ListPublishedNotes)
		notes.PATCH(":id", h.UpdateNote)
		notes.DELETE(":id", h.DeleteNote)
		notes.POST(":id/restore", h.RestoreNote)
		notes.POST(":id/comments", h.CreateNoteComment)
		notes.GET(":id/comments", h.ListNoteComments)

		aiStudent := api.Group("/ai", studentGuard)
		aiStudent.GET("/usage", h.GetAIUsageSummary)
		aiStudent.GET("/sessions", h.ListAIChatSessions)
		aiStudent.POST("/sessions", h.CreateAIChatSession)
		aiStudent.PATCH("/sessions/:id", h.UpdateAIChatSession)
		aiStudent.DELETE("/sessions/:id", h.DeleteAIChatSession)
		aiStudent.POST("/sessions/:id/close", h.CloseAIChatSession)
		aiStudent.GET("/sessions/:id/messages", h.ListAIChatMessages)
		aiStudent.POST("/sessions/:id/messages", h.SendAIChatMessage)
		aiStudent.POST("/check_assignment", h.CheckAssignment)

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

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type passwordResetRequest struct {
	SchoolID   string `json:"school_id" validate:"required"`
	Identifier string `json:"identifier" validate:"required"`
}

type passwordResetConfirmRequest struct {
	SchoolID    string `json:"school_id" validate:"required"`
	Identifier  string `json:"identifier" validate:"required"`
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
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

func (h *Handler) RefreshToken(c *gin.Context) {
	var req refreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	access, refresh, account, err := h.auth.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRefreshToken):
			response.Error(c, http.StatusUnauthorized, "invalid refresh token", nil)
		case errors.Is(err, service.ErrAccountLocked):
			response.Error(c, http.StatusForbidden, "account locked", nil)
		case errors.Is(err, service.ErrPasswordResetRequired):
			response.Error(c, http.StatusForbidden, "password reset required", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "refresh failed", err.Error())
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

func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req passwordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	token, expiresAt, err := h.auth.RequestPasswordReset(c.Request.Context(), req.SchoolID, req.Identifier)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Error(c, http.StatusUnauthorized, "invalid credentials", nil)
		case errors.Is(err, service.ErrPasswordResetUnavailable):
			response.Error(c, http.StatusBadRequest, "password reset unavailable", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to issue reset token", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"reset_token": token,
		"expires_at":  expiresAt,
	})
}

func (h *Handler) ConfirmPasswordReset(c *gin.Context) {
	var req passwordResetConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	if err := h.auth.ResetPassword(c.Request.Context(), req.SchoolID, req.Identifier, req.Token, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Error(c, http.StatusUnauthorized, "invalid credentials", nil)
		case errors.Is(err, service.ErrPasswordResetUnavailable):
			response.Error(c, http.StatusBadRequest, "password reset unavailable", nil)
		case errors.Is(err, service.ErrPasswordResetTokenInvalid):
			response.Error(c, http.StatusBadRequest, "invalid reset token", nil)
		case errors.Is(err, service.ErrPasswordResetTokenExpired):
			response.Error(c, http.StatusBadRequest, "reset token expired", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to reset password", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"status": "password_updated"})
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
	SchoolID   string `json:"school_id" validate:"required"`
	Number     string `json:"number" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Email      string `json:"email" validate:"required,email"`
	Phone      string `json:"phone"`
	ClassID    string `json:"class_id"`
	DefaultPwd string `json:"default_password" validate:"required"`
}

type batchAccountActionRequest struct {
	SchoolID   string   `json:"school_id" validate:"required"`
	AccountIDs []string `json:"account_ids" validate:"required,min=1,dive,required"`
	Action     string   `json:"action" validate:"required"`
}

type createStudentReminderRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
	TimeLabel   string `json:"time_label"`
	Route       string `json:"route"`
	Priority    string `json:"priority" validate:"omitempty,oneof=normal high"`
	Icon        string `json:"icon"`
}

type updateStudentReminderRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	TimeLabel   *string `json:"time_label"`
	Route       *string `json:"route"`
	Priority    *string `json:"priority"`
	Icon        *string `json:"icon"`
	Completed   *bool   `json:"completed"`
}

type reminderCompletionRequest struct {
	Completed *bool `json:"completed"`
}

type batchReminderCompletionRequest struct {
	ReminderIDs []string `json:"reminder_ids" validate:"required,min=1,dive,required"`
	Completed   *bool    `json:"completed"`
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
	courseID := strings.TrimSpace(c.Query("course_id"))
	// if classID != "" && departmentID == "" {
	// 	response.Error(c, http.StatusBadRequest, "department_id required when class_id provided", nil)
	// 	return
	// }

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
		CourseID:        courseID,
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

type updateAIAgentSettingRequest struct {
	SchoolID                string  `json:"school_id" validate:"required"`
	Provider                string  `json:"provider" validate:"required"`
	Model                   string  `json:"model" validate:"required"`
	APIKey                  string  `json:"api_key"`
	BaseURL                 string  `json:"base_url"`
	Temperature             float32 `json:"temperature" validate:"required"`
	TopP                    float32 `json:"top_p" validate:"required"`
	MaxOutputTokens         int     `json:"max_output_tokens" validate:"required"`
	MaxDailyRequests        int     `json:"max_daily_requests" validate:"required"`
	MaxConcurrentRequests   int     `json:"max_concurrent_requests" validate:"required"`
	MaxConversationMessages int     `json:"max_conversation_messages" validate:"required"`
	SystemPrompt            string  `json:"system_prompt"`
	VisionEnabled           bool    `json:"vision_enabled"`
}

type createAIChatSessionRequest struct {
	Title string `json:"title"`
}

type updateAIChatSessionRequest struct {
	Title string `json:"title" validate:"required"`
}

type sendAIChatMessageRequest struct {
	Content string `json:"content" validate:"required"`
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

func (h *Handler) UpdateAccountStructure(c *gin.Context) {
	accountID := strings.TrimSpace(c.Param("id"))
	if accountID == "" {
		response.Error(c, http.StatusBadRequest, "account id is required", nil)
		return
	}

	type updateRequest struct {
		ClassID *string `json:"class_id"`
	}
	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	err := h.admin.UpdateAccountStructure(c.Request.Context(), service.UpdateAccountStructureInput{
		SchoolID:  schoolID,
		AccountID: accountID,
		ClassID:   req.ClassID,
	})

	if err != nil {
		h.handleAdminAccountError(c, err, "unable to update account structure")
		return
	}

	response.Success(c, http.StatusOK, nil)
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

func (h *Handler) BatchOperateAccounts(c *gin.Context) {
	var req batchAccountActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	result, err := h.admin.BatchOperateAccounts(c.Request.Context(), service.AdminBatchOperationInput{
		SchoolID:   strings.TrimSpace(req.SchoolID),
		AccountIDs: req.AccountIDs,
		Action:     service.AdminBatchAction(strings.ToLower(strings.TrimSpace(req.Action))),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAdminBatchAccountIDsRequired):
			response.Error(c, http.StatusBadRequest, "account_ids required", nil)
		case errors.Is(err, service.ErrAdminBatchActionUnsupported):
			response.Error(c, http.StatusBadRequest, "unsupported action", nil)
		default:
			response.Error(c, http.StatusBadRequest, "unable to complete batch operation", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"succeeded": result.Succeeded,
		"failed":    result.Failed,
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

func (h *Handler) GetAIAgentSetting(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	setting, err := h.ai.GetSetting(c.Request.Context(), schoolID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to load ai setting", err.Error())
		return
	}

	if setting == nil {
		response.Success(c, http.StatusOK, gin.H{"setting": nil})
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"setting": gin.H{
			"id":                        setting.ID,
			"school_id":                 setting.SchoolID,
			"provider":                  setting.Provider,
			"model":                     setting.Model,
			"base_url":                  setting.BaseURL,
			"temperature":               setting.Temperature,
			"top_p":                     setting.TopP,
			"max_output_tokens":         setting.MaxOutputTokens,
			"max_daily_requests":        setting.MaxDailyRequests,
			"max_concurrent_requests":   setting.MaxConcurrentRequests,
			"max_conversation_messages": setting.MaxConversationMessages,
			"system_prompt":             setting.SystemPrompt,
			"vision_enabled":            setting.VisionEnabled,
			"updated_by":                setting.UpdatedBy,
			"updated_by_name":           setting.UpdatedByName,
			"updated_at":                setting.UpdatedAt,
			"created_at":                setting.CreatedAt,
			"api_key_present":           setting.APIKey != "",
		},
	})
}

func (h *Handler) UpdateAIAgentSetting(c *gin.Context) {
	accountIDValue, exists := c.Get(middleware.ContextAccountID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}
	accountID, _ := accountIDValue.(string)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "invalid account context", nil)
		return
	}

	var req updateAIAgentSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	setting, err := h.ai.UpdateSetting(c.Request.Context(), service.UpdateAIAgentSettingInput{
		SchoolID:                strings.TrimSpace(req.SchoolID),
		Provider:                domain.AIProvider(strings.TrimSpace(strings.ToLower(req.Provider))),
		Model:                   strings.TrimSpace(req.Model),
		APIKey:                  strings.TrimSpace(req.APIKey),
		BaseURL:                 strings.TrimSpace(req.BaseURL),
		Temperature:             req.Temperature,
		TopP:                    req.TopP,
		MaxOutputTokens:         req.MaxOutputTokens,
		MaxDailyRequests:        req.MaxDailyRequests,
		MaxConcurrentRequests:   req.MaxConcurrentRequests,
		MaxConversationMessages: req.MaxConversationMessages,
		SystemPrompt:            req.SystemPrompt,
		VisionEnabled:           req.VisionEnabled,
		OperatorID:              accountID,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "unsupported provider"):
			response.Error(c, http.StatusBadRequest, "unsupported provider", err.Error())
		case strings.Contains(err.Error(), "required"):
			response.Error(c, http.StatusBadRequest, "invalid configuration", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "unable to update ai setting", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{
		"setting": gin.H{
			"id":                        setting.ID,
			"school_id":                 setting.SchoolID,
			"provider":                  setting.Provider,
			"model":                     setting.Model,
			"base_url":                  setting.BaseURL,
			"temperature":               setting.Temperature,
			"top_p":                     setting.TopP,
			"max_output_tokens":         setting.MaxOutputTokens,
			"max_daily_requests":        setting.MaxDailyRequests,
			"max_concurrent_requests":   setting.MaxConcurrentRequests,
			"max_conversation_messages": setting.MaxConversationMessages,
			"system_prompt":             setting.SystemPrompt,
			"vision_enabled":            setting.VisionEnabled,
			"updated_by":                setting.UpdatedBy,
			"updated_by_name":           setting.UpdatedByName,
			"updated_at":                setting.UpdatedAt,
			"api_key_present":           setting.APIKey != "",
		},
	})
}

func (h *Handler) ListAIAgentSettingAudits(c *gin.Context) {
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

	entries, err := h.ai.ListSettingAudits(c.Request.Context(), schoolID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list ai audit logs", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"entries": entries})
}

func (h *Handler) ListAIUsageSummaries(c *gin.Context) {
	usageInput, ok := h.buildUsageListInput(c)
	if !ok {
		return
	}

	page := usageInput.Page
	pageSize := usageInput.PageSize

	summaries, err := h.ai.ListUsageSummaries(c.Request.Context(), usageInput)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch ai usage", err.Error())
		}
		return
	}

	aggregate, err := h.ai.GetUsageAggregate(c.Request.Context(), usageInput)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch ai usage", err.Error())
		}
		return
	}

	roleBreakdown, err := h.ai.GetUsageRoleBreakdown(c.Request.Context(), usageInput)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch ai usage", err.Error())
		}
		return
	}

	items := make([]gin.H, 0, len(summaries))
	for _, summary := range summaries {
		items = append(items, aiUsageSummaryPayload(summary))
	}

	payload := gin.H{"usage": items}
	if aggregate != nil {
		payload["overview"] = aiUsageAggregatePayload(*aggregate, roleBreakdown)
	}
	totalAccounts := 0
	if aggregate != nil {
		totalAccounts = aggregate.ActiveAccounts
	}
	payload["pagination"] = gin.H{
		"page":           page,
		"page_size":      pageSize,
		"returned":       len(items),
		"has_more":       len(items) == pageSize,
		"total_accounts": totalAccounts,
	}

	response.Success(c, http.StatusOK, payload)
}

func (h *Handler) buildUsageListInput(c *gin.Context) (service.ListUsageSummariesInput, bool) {
	var input service.ListUsageSummariesInput

	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return input, false
	}
	input.SchoolID = schoolID

	if sinceParam := strings.TrimSpace(c.Query("since")); sinceParam != "" {
		parsed, err := time.Parse(time.RFC3339, sinceParam)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid since timestamp", err.Error())
			return input, false
		}
		input.Since = parsed
	}

	if roleParam := strings.TrimSpace(c.Query("role")); roleParam != "" {
		input.Role = domain.Role(roleParam)
	}

	if minUserMessagesParam := strings.TrimSpace(c.Query("min_user_messages")); minUserMessagesParam != "" {
		value, err := strconv.ParseInt(minUserMessagesParam, 10, 64)
		if err != nil || value < 0 {
			response.Error(c, http.StatusBadRequest, "invalid min_user_messages", nil)
			return input, false
		}
		input.MinUserMessages = value
	}

	if minTotalMessagesParam := strings.TrimSpace(c.Query("min_total_messages")); minTotalMessagesParam != "" {
		value, err := strconv.ParseInt(minTotalMessagesParam, 10, 64)
		if err != nil || value < 0 {
			response.Error(c, http.StatusBadRequest, "invalid min_total_messages", nil)
			return input, false
		}
		input.MinTotalMessages = value
	}

	pageSize := service.DefaultUsageSummaryPageSize
	if limitParam := strings.TrimSpace(c.Query("limit")); limitParam != "" {
		value, err := strconv.Atoi(limitParam)
		if err != nil || value <= 0 {
			response.Error(c, http.StatusBadRequest, "invalid limit", nil)
			return input, false
		}
		pageSize = value
	}

	if pageSizeParam := strings.TrimSpace(c.Query("page_size")); pageSizeParam != "" {
		value, err := strconv.Atoi(pageSizeParam)
		if err != nil || value <= 0 {
			response.Error(c, http.StatusBadRequest, "invalid page_size", nil)
			return input, false
		}
		pageSize = value
	}
	input.PageSize = pageSize

	page := 1
	if pageParam := strings.TrimSpace(c.Query("page")); pageParam != "" {
		value, err := strconv.Atoi(pageParam)
		if err != nil || value <= 0 {
			response.Error(c, http.StatusBadRequest, "invalid page", nil)
			return input, false
		}
		page = value
	}
	input.Page = page

	sortField := service.UsageSortFieldUserMessages
	if sortByParam := strings.ToLower(strings.TrimSpace(c.Query("sort_by"))); sortByParam != "" {
		switch sortByParam {
		case "user_messages":
			sortField = service.UsageSortFieldUserMessages
		case "total_messages":
			sortField = service.UsageSortFieldTotalMessages
		case "total_tokens":
			sortField = service.UsageSortFieldTotalTokens
		default:
			response.Error(c, http.StatusBadRequest, "invalid sort_by", nil)
			return input, false
		}
	}
	input.SortField = sortField

	sortDirection := service.UsageSortDirectionDesc
	if sortDirParam := strings.ToLower(strings.TrimSpace(c.Query("sort_dir"))); sortDirParam != "" {
		switch sortDirParam {
		case "asc":
			sortDirection = service.UsageSortDirectionAsc
		case "desc":
			sortDirection = service.UsageSortDirectionDesc
		default:
			response.Error(c, http.StatusBadRequest, "invalid sort_dir", nil)
			return input, false
		}
	}
	input.SortDirection = sortDirection

	return input, true
}

func (h *Handler) ExportAIUsageSummaries(c *gin.Context) {
	usageInput, ok := h.buildUsageListInput(c)
	if !ok {
		return
	}
	usageInput.Page = 1

	summaries, err := h.ai.ListAllUsageSummaries(c.Request.Context(), usageInput)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to export ai usage", err.Error())
		}
		return
	}

	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	headers := []string{
		"account_id",
		"account_name",
		"role",
		"user_messages",
		"assistant_messages",
		"total_messages",
		"prompt_tokens",
		"result_tokens",
		"total_tokens",
		"max_daily_requests",
		"remaining_daily_requests",
	}
	if err := writer.Write(headers); err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to export ai usage", err.Error())
		return
	}

	for _, summary := range summaries {
		record := []string{
			summary.AccountID,
			summary.AccountName,
			string(summary.Role),
			strconv.FormatInt(summary.UserMessages, 10),
			strconv.FormatInt(summary.AssistantMessages, 10),
			strconv.FormatInt(summary.TotalMessages, 10),
			strconv.FormatInt(summary.PromptTokens, 10),
			strconv.FormatInt(summary.ResultTokens, 10),
			strconv.FormatInt(summary.TotalTokens, 10),
			strconv.Itoa(summary.MaxDailyRequests),
			strconv.Itoa(summary.RemainingDailyRequests),
		}
		if err := writer.Write(record); err != nil {
			response.Error(c, http.StatusInternalServerError, "unable to export ai usage", err.Error())
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to export ai usage", err.Error())
		return
	}

	filename := fmt.Sprintf("ai-usage-%s-%s.csv", usageInput.SchoolID, time.Now().UTC().Format("20060102T150405Z"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func (h *Handler) GetAIUsageReport(c *gin.Context) {
	usageInput, ok := h.buildUsageListInput(c)
	if !ok {
		return
	}

	summaries, err := h.ai.ListUsageSummaries(c.Request.Context(), usageInput)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to build ai usage report", err.Error())
		}
		return
	}

	aggregate, err := h.ai.GetUsageAggregate(c.Request.Context(), usageInput)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to build ai usage report", err.Error())
		}
		return
	}

	roleBreakdown, err := h.ai.GetUsageRoleBreakdown(c.Request.Context(), usageInput)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to build ai usage report", err.Error())
		}
		return
	}

	items := make([]gin.H, 0, len(summaries))
	for _, summary := range summaries {
		items = append(items, aiUsageSummaryPayload(summary))
	}

	report := gin.H{
		"generated_at": time.Now(),
		"filters": gin.H{
			"school_id":          usageInput.SchoolID,
			"role":               usageInput.Role,
			"since":              usageInput.Since,
			"min_user_messages":  usageInput.MinUserMessages,
			"min_total_messages": usageInput.MinTotalMessages,
			"sort_by":            usageInput.SortField,
			"sort_dir":           usageInput.SortDirection,
		},
		"accounts": items,
	}

	if aggregate != nil {
		report["overview"] = aiUsageAggregatePayload(*aggregate, roleBreakdown)
		metrics := gin.H{
			"active_accounts": aggregate.ActiveAccounts,
			"total_messages":  aggregate.TotalMessages,
			"total_tokens":    aggregate.TotalTokens,
		}
		if aggregate.ActiveAccounts > 0 {
			metrics["avg_messages_per_account"] = float64(aggregate.TotalMessages) / float64(aggregate.ActiveAccounts)
			metrics["avg_tokens_per_account"] = float64(aggregate.TotalTokens) / float64(aggregate.ActiveAccounts)
		}
		report["metrics"] = metrics
	} else {
		report["metrics"] = gin.H{}
	}

	highlights := gin.H{}
	if len(summaries) > 0 {
		top := summaries[0]
		highlights["top_account"] = gin.H{
			"account_id":     top.AccountID,
			"account_name":   top.AccountName,
			"role":           top.Role,
			"total_messages": top.TotalMessages,
			"total_tokens":   top.TotalTokens,
		}
		if aggregate != nil && aggregate.TotalMessages > 0 {
			highlights["top_account_share"] = float64(top.TotalMessages) / float64(aggregate.TotalMessages)
		}
	}
	report["highlights"] = highlights

	response.Success(c, http.StatusOK, gin.H{"report": report})
}

func (h *Handler) GetAIUsageTimeline(c *gin.Context) {
	schoolID := strings.TrimSpace(c.Query("school_id"))
	if schoolID == "" {
		response.Error(c, http.StatusBadRequest, "school_id required", nil)
		return
	}

	roleParam := strings.TrimSpace(c.Query("role"))
	var role domain.Role
	if roleParam != "" {
		role = domain.Role(roleParam)
	}

	parseTime := func(value string, field string) (time.Time, bool) {
		value = strings.TrimSpace(value)
		if value == "" {
			return time.Time{}, true
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid "+field, err.Error())
			return time.Time{}, false
		}
		return parsed, true
	}

	start, ok := parseTime(c.Query("start"), "start timestamp")
	if !ok {
		return
	}
	end, ok := parseTime(c.Query("end"), "end timestamp")
	if !ok {
		return
	}

	windowParam := strings.TrimSpace(c.Query("window_days"))
	var windowDays int
	if windowParam != "" {
		value, err := strconv.Atoi(windowParam)
		if err != nil || value <= 0 {
			response.Error(c, http.StatusBadRequest, "invalid window_days", nil)
			return
		}
		windowDays = value
	}

	intervalParam := strings.TrimSpace(strings.ToLower(c.Query("interval")))
	interval := service.UsageTimelineIntervalDay
	if intervalParam != "" {
		switch intervalParam {
		case "day", "daily":
			interval = service.UsageTimelineIntervalDay
		default:
			response.Error(c, http.StatusBadRequest, "invalid interval", intervalParam)
			return
		}
	}

	input := service.UsageTimelineInput{
		SchoolID:   schoolID,
		Role:       role,
		Start:      start,
		End:        end,
		Interval:   interval,
		WindowDays: windowDays,
	}

	points, err := h.ai.GetUsageTimeline(c.Request.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		case errors.Is(err, service.ErrUsageTimelineUnsupportedInterval), errors.Is(err, service.ErrUsageTimelineInvalidRange):
			response.Error(c, http.StatusBadRequest, "invalid timeline request", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch ai timeline", err.Error())
		}
		return
	}

	payload := gin.H{
		"timeline": aiUsageTimelinePayload(points),
		"interval": string(interval),
	}
	if len(points) > 0 {
		payload["start"] = points[0].Date
		payload["end"] = points[len(points)-1].Date
	}

	response.Success(c, http.StatusOK, payload)
}

func (h *Handler) CreateAIChatSession(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	var req createAIChatSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if !errors.Is(err, io.EOF) {
			response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
			return
		}
	}

	session, err := h.ai.CreateSession(c.Request.Context(), service.CreateAIChatSessionInput{
		AccountID: accountID,
		Title:     strings.TrimSpace(req.Title),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAIAccountNotFound):
			response.Error(c, http.StatusUnauthorized, "account not found", nil)
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to create ai session", err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"session": aiChatSessionPayload(*session)})
}

func (h *Handler) GetAIUsageSummary(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	sinceParam := strings.TrimSpace(c.Query("since"))
	var since time.Time
	if sinceParam != "" {
		parsed, err := time.Parse(time.RFC3339, sinceParam)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid since timestamp", err.Error())
			return
		}
		since = parsed
	}

	summary, err := h.ai.GetUsageSummary(c.Request.Context(), accountID, since)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAIAccountNotFound):
			response.Error(c, http.StatusUnauthorized, "account not found", nil)
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch ai usage", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"usage": aiUsageSummaryPayload(*summary)})
}

func (h *Handler) CloseAIChatSession(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	if sessionID == "" {
		response.Error(c, http.StatusBadRequest, "session_id required", nil)
		return
	}

	session, err := h.ai.CloseSession(c.Request.Context(), accountID, sessionID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAIAccountNotFound):
			response.Error(c, http.StatusUnauthorized, "account not found", nil)
		case errors.Is(err, service.ErrAIChatSessionNotFound):
			response.Error(c, http.StatusNotFound, "session not found", nil)
		case errors.Is(err, service.ErrAIChatSessionForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to access session", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to close ai session", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"session": aiChatSessionPayload(*session)})
}

func (h *Handler) UpdateAIChatSession(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	if sessionID == "" {
		response.Error(c, http.StatusBadRequest, "session_id required", nil)
		return
	}

	var req updateAIChatSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	session, err := h.ai.UpdateSessionTitle(c.Request.Context(), accountID, sessionID, req.Title)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAIAccountNotFound):
			response.Error(c, http.StatusUnauthorized, "account not found", nil)
		case errors.Is(err, service.ErrAIChatSessionNotFound):
			response.Error(c, http.StatusNotFound, "session not found", nil)
		case errors.Is(err, service.ErrAIChatSessionForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to access session", nil)
		case errors.Is(err, service.ErrAIChatSessionTitleEmpty):
			response.Error(c, http.StatusBadRequest, "title required", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to update ai session", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"session": aiChatSessionPayload(*session)})
}

func (h *Handler) DeleteAIChatSession(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	if sessionID == "" {
		response.Error(c, http.StatusBadRequest, "session_id required", nil)
		return
	}

	if err := h.ai.DeleteSession(c.Request.Context(), accountID, sessionID); err != nil {
		switch {
		case errors.Is(err, service.ErrAIAccountNotFound):
			response.Error(c, http.StatusUnauthorized, "account not found", nil)
		case errors.Is(err, service.ErrAIChatSessionNotFound):
			response.Error(c, http.StatusNotFound, "session not found", nil)
		case errors.Is(err, service.ErrAIChatSessionForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to access session", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to delete ai session", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, nil)
}

func (h *Handler) ListAIChatSessions(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	limitParam := strings.TrimSpace(c.DefaultQuery("limit", "10"))
	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 {
		limit = 10
	}

	sessions, err := h.ai.ListSessions(c.Request.Context(), accountID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list ai sessions", err.Error())
		return
	}

	items := make([]gin.H, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, aiChatSessionPayload(session))
	}

	response.Success(c, http.StatusOK, gin.H{"sessions": items})
}

func (h *Handler) ListAIChatMessages(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	if sessionID == "" {
		response.Error(c, http.StatusBadRequest, "session_id required", nil)
		return
	}

	limitParam := strings.TrimSpace(c.DefaultQuery("limit", "20"))
	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 {
		limit = 20
	}

	var before time.Time
	if beforeRaw := strings.TrimSpace(c.Query("before")); beforeRaw != "" {
		parsed, parseErr := time.Parse(time.RFC3339, beforeRaw)
		if parseErr != nil {
			response.Error(c, http.StatusBadRequest, "invalid before timestamp", parseErr.Error())
			return
		}
		before = parsed
	}

	messages, err := h.ai.ListSessionMessages(c.Request.Context(), accountID, sessionID, limit, before)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAIChatSessionNotFound):
			response.Error(c, http.StatusNotFound, "session not found", nil)
		case errors.Is(err, service.ErrAIChatSessionForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to view session", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to list ai messages", err.Error())
		}
		return
	}

	items := make([]gin.H, 0, len(messages))
	for _, message := range messages {
		items = append(items, aiChatMessagePayload(message))
	}

	response.Success(c, http.StatusOK, gin.H{
		"session_id": sessionID,
		"messages":   items,
	})
}

func (h *Handler) SendAIChatMessage(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	sessionID := strings.TrimSpace(c.Param("id"))
	if sessionID == "" {
		response.Error(c, http.StatusBadRequest, "session_id required", nil)
		return
	}

	var req sendAIChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	result, err := h.ai.SendMessage(c.Request.Context(), service.SendAIChatMessageInput{
		AccountID: accountID,
		SessionID: sessionID,
		Content:   req.Content,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAIAccountNotFound):
			response.Error(c, http.StatusUnauthorized, "account not found", nil)
		case errors.Is(err, service.ErrAIChatSessionNotFound):
			response.Error(c, http.StatusNotFound, "session not found", nil)
		case errors.Is(err, service.ErrAIChatSessionForbidden):
			response.Error(c, http.StatusForbidden, "not allowed to access session", nil)
		case errors.Is(err, service.ErrAISettingNotConfigured):
			response.Error(c, http.StatusConflict, "ai assistant not configured", nil)
		case errors.Is(err, service.ErrAIChatSessionLimitReached):
			response.Error(c, http.StatusConflict, "session turn limit reached", nil)
		case errors.Is(err, service.ErrAIChatSessionClosed):
			response.Error(c, http.StatusConflict, "session closed", nil)
		case errors.Is(err, service.ErrAIChatDailyLimitExceeded):
			response.Error(c, http.StatusTooManyRequests, "daily limit reached", nil)
		case errors.Is(err, service.ErrAIChatConcurrentLimitReached):
			response.Error(c, http.StatusTooManyRequests, "too many concurrent requests", nil)
		case errors.Is(err, service.ErrAIChatMessageEmpty):
			response.Error(c, http.StatusBadRequest, "content required", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to send ai message", err.Error())
		}
		return
	}

	status := "pending"
	respPayload := gin.H{
		"session":      aiChatSessionPayload(*result.Session),
		"user_message": aiChatMessagePayload(*result.UserMessage),
	}
	if result.AssistantMessage != nil {
		status = "completed"
		respPayload["assistant_message"] = aiChatMessagePayload(*result.AssistantMessage)
	}
	if result.ProviderError != nil {
		respPayload["provider_error"] = result.ProviderError.Error()
	}
	if result.ProviderReason != "" {
		respPayload["provider_reason"] = result.ProviderReason
	}
	respPayload["status"] = status

	response.Success(c, http.StatusCreated, respPayload)
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
	switches, err := h.system.ListSwitches(c.Request.Context(), schoolID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list switches", err.Error())
		return
	}
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

	updated, err := h.system.UpdateSwitchState(c.Request.Context(), req.SchoolID, switchID, *req.Enabled, "")
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
	params, err := h.system.ListParameters(c.Request.Context(), schoolID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list parameters", err.Error())
		return
	}
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

	updated, err := h.system.UpdateParameter(c.Request.Context(), req.SchoolID, parameterID, req.Value, "")
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
	items, err := h.system.ListBroadcasts(c.Request.Context(), schoolID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list broadcasts", err.Error())
		return
	}
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

	updated, err := h.system.UpdateBroadcast(c.Request.Context(), req.SchoolID, broadcastID, statusPtr, req.Pinned, "")
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
	logs, err := h.system.ListAuditLogs(c.Request.Context(), schoolID, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list audit logs", err.Error())
		return
	}
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
			"id":            dept.ID,
			"school_id":     dept.SchoolID,
			"name":          dept.Name,
			"created_at":    dept.CreatedAt,
			"updated_at":    dept.UpdatedAt,
			"teacher_count": dept.TeacherCount,
			"student_count": dept.StudentCount,
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
			"student_count": class.StudentCount,
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

type updateAssignmentRequest struct {
	TeacherID     string               `json:"teacher_id" validate:"required"`
	Title         *string              `json:"title"`
	Description   *string              `json:"description"`
	StartAt       *service.TimeISO8601 `json:"start_at"`
	DueAt         *service.TimeISO8601 `json:"due_at"`
	MaxScore      *float64             `json:"max_score"`
	AllowResubmit *bool                `json:"allow_resubmit"`
}

func (h *Handler) UpdateAssignment(c *gin.Context) {
	assignmentID := strings.TrimSpace(c.Param("id"))
	if assignmentID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment id", nil)
		return
	}

	var req updateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	startAt := convertToTime(req.StartAt)
	dueAt := convertToTime(req.DueAt)

	assignment, err := h.assignments.UpdateAssignment(c.Request.Context(), service.UpdateAssignmentInput{
		ID:            assignmentID,
		TeacherID:     req.TeacherID,
		Title:         req.Title,
		Description:   req.Description,
		StartAt:       startAt,
		DueAt:         dueAt,
		MaxScore:      req.MaxScore,
		AllowResubmit: req.AllowResubmit,
	})
	if err != nil {
		response.Error(c, http.StatusBadRequest, "unable to update assignment", err.Error())
		return
	}

	response.Success(c, http.StatusOK, gin.H{"assignment_id": assignment.ID})
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

func (h *Handler) ListStudentAssignments(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	limit := parseLimit(c.DefaultQuery("limit", "20"), 20, 200)
	items, err := h.student.ListAssignments(c.Request.Context(), accountID, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch assignments", err.Error())
		}
		return
	}

	payload := make([]gin.H, 0, len(items))
	for _, item := range items {
		payload = append(payload, gin.H{
			"id":             item.ID,
			"title":          item.Title,
			"description":    item.Description,
			"course_id":      item.CourseID,
			"course_name":    item.CourseName,
			"teacher_id":     item.TeacherID,
			"teacher_name":   item.TeacherName,
			"type":           string(item.Type),
			"start_at":       item.StartAt,
			"due_at":         item.DueAt,
			"allow_resubmit": item.AllowResubmit,
			"status":         item.Status,
			"submitted_at":   item.SubmittedAt,
			"score":          item.Score,
			"feedback":       item.Feedback,
			"is_overdue":     item.IsOverdue,
		})
	}

	response.Success(c, http.StatusOK, gin.H{"assignments": payload})
}

func (h *Handler) ListStudentExams(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	limit := parseLimit(c.DefaultQuery("limit", "20"), 20, 200)
	items, err := h.student.ListExams(c.Request.Context(), accountID, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch exams", err.Error())
		}
		return
	}

	payload := make([]gin.H, 0, len(items))
	for _, item := range items {
		payload = append(payload, gin.H{
			"id":             item.ID,
			"title":          item.Title,
			"description":    item.Description,
			"course_id":      item.CourseID,
			"course_name":    item.CourseName,
			"teacher_id":     item.TeacherID,
			"teacher_name":   item.TeacherName,
			"type":           string(item.Type),
			"start_at":       item.StartAt,
			"due_at":         item.DueAt,
			"allow_resubmit": item.AllowResubmit,
			"status":         item.Status,
			"submitted_at":   item.SubmittedAt,
			"score":          item.Score,
			"feedback":       item.Feedback,
			"is_overdue":     item.IsOverdue,
		})
	}

	response.Success(c, http.StatusOK, gin.H{"exams": payload})
}

func (h *Handler) ListStudentSchedule(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	start, end, err := parseScheduleRange(c.Query("from"), c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid time range", err.Error())
		return
	}

	items, err := h.student.ListSchedule(c.Request.Context(), accountID, start, end)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch schedule", err.Error())
		}
		return
	}

	payload := make([]gin.H, 0, len(items))
	for _, item := range items {
		payload = append(payload, gin.H{
			"session_id":   item.SessionID,
			"course_id":    item.CourseID,
			"course_name":  item.CourseName,
			"teacher_id":   item.TeacherID,
			"teacher_name": item.TeacherName,
			"starts_at":    item.StartsAt,
			"ends_at":      item.EndsAt,
			"day":          item.Day,
			"slot_id":      item.SlotID,
			"slot_name":    item.SlotName,
			"location":     item.Location,
			"source":       item.Source,
		})
	}

	response.Success(c, http.StatusOK, gin.H{"sessions": payload})
}

func (h *Handler) ListStudentAgenda(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	start, end, err := parseScheduleRange(c.Query("from"), c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid time range", err.Error())
		return
	}
	includeAssignments := parseBool(c.DefaultQuery("include_assignments", "true"), true)

	items, err := h.student.ListAgenda(c.Request.Context(), accountID, start, end, includeAssignments)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch agenda", err.Error())
		}
		return
	}

	agenda := make([]gin.H, 0, len(items))
	for _, item := range items {
		entry := gin.H{
			"id":           item.ID,
			"kind":         string(item.Kind),
			"title":        item.Title,
			"description":  item.Description,
			"course_id":    item.CourseID,
			"course_name":  item.CourseName,
			"teacher_id":   item.TeacherID,
			"teacher_name": item.TeacherName,
			"starts_at":    item.StartsAt,
			"day":          item.Day,
			"location":     item.Location,
			"source":       item.Source,
			"status":       item.Status,
			"is_overdue":   item.IsOverdue,
			"feedback":     item.Feedback,
		}
		if item.EndsAt != nil {
			entry["ends_at"] = item.EndsAt
		}
		if item.SubmittedAt != nil {
			entry["submitted_at"] = item.SubmittedAt
		}
		if item.Score != nil {
			entry["score"] = item.Score
		}
		agenda = append(agenda, entry)
	}

	response.Success(c, http.StatusOK, gin.H{"agenda": agenda})
}

func (h *Handler) ListStudentReminders(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	items, err := h.student.ListCustomReminders(c.Request.Context(), accountID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch reminders", err.Error())
		}
		return
	}

	reminders := make([]gin.H, 0, len(items))
	for _, item := range items {
		reminders = append(reminders, studentReminderPayload(item))
	}

	response.Success(c, http.StatusOK, gin.H{"reminders": reminders})
}

func (h *Handler) CreateStudentReminder(c *gin.Context) {
	var req createStudentReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	priority := parseReminderPriority(req.Priority)
	reminder, err := h.student.CreateCustomReminder(c.Request.Context(), accountID, service.CreateStudentReminderInput{
		Title:       req.Title,
		Description: req.Description,
		TimeLabel:   req.TimeLabel,
		Route:       req.Route,
		Priority:    priority,
		Icon:        req.Icon,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		case errors.Is(err, service.ErrStudentReminderInvalid):
			response.Error(c, http.StatusBadRequest, "invalid reminder", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to create reminder", err.Error())
		}
		return
	}

	response.Success(c, http.StatusCreated, gin.H{"reminder": studentReminderPayload(*reminder)})
}

func (h *Handler) UpdateStudentReminder(c *gin.Context) {
	reminderID := strings.TrimSpace(c.Param("id"))
	if reminderID == "" {
		response.Error(c, http.StatusBadRequest, "missing reminder id", nil)
		return
	}

	var req updateStudentReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	input := service.UpdateStudentReminderInput{}
	if req.Title != nil {
		input.Title = req.Title
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.TimeLabel != nil {
		input.TimeLabel = req.TimeLabel
	}
	if req.Route != nil {
		input.Route = req.Route
	}
	if req.Priority != nil {
		priority := parseReminderPriority(*req.Priority)
		input.Priority = &priority
	}
	if req.Icon != nil {
		input.Icon = req.Icon
	}
	if req.Completed != nil {
		input.Completed = req.Completed
	}

	reminder, err := h.student.UpdateCustomReminder(c.Request.Context(), accountID, reminderID, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		case errors.Is(err, service.ErrStudentReminderNotFound):
			response.Error(c, http.StatusNotFound, "reminder not found", nil)
		case errors.Is(err, service.ErrStudentReminderInvalid):
			response.Error(c, http.StatusBadRequest, "invalid reminder", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to update reminder", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"reminder": studentReminderPayload(*reminder)})
}

func (h *Handler) DeleteStudentReminder(c *gin.Context) {
	reminderID := strings.TrimSpace(c.Param("id"))
	if reminderID == "" {
		response.Error(c, http.StatusBadRequest, "missing reminder id", nil)
		return
	}

	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	if err := h.student.DeleteCustomReminder(c.Request.Context(), accountID, reminderID); err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		case errors.Is(err, service.ErrStudentReminderNotFound):
			response.Error(c, http.StatusNotFound, "reminder not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to delete reminder", err.Error())
		}
		return
	}

	response.Success(c, http.StatusNoContent, nil)
}

func (h *Handler) UpdateStudentReminderCompletion(c *gin.Context) {
	reminderID := strings.TrimSpace(c.Param("id"))
	if reminderID == "" {
		response.Error(c, http.StatusBadRequest, "missing reminder id", nil)
		return
	}

	var req reminderCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	completed := true
	if req.Completed != nil {
		completed = *req.Completed
	}

	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	reminder, err := h.student.SetReminderCompletion(c.Request.Context(), accountID, reminderID, completed)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		case errors.Is(err, service.ErrStudentReminderNotFound):
			response.Error(c, http.StatusNotFound, "reminder not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to update reminder", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"reminder": studentReminderPayload(*reminder)})
}

func (h *Handler) BatchUpdateStudentReminderCompletion(c *gin.Context) {
	var req batchReminderCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation error", err.Error())
		return
	}

	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	ids := make([]string, 0, len(req.ReminderIDs))
	for _, id := range req.ReminderIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		ids = append(ids, trimmed)
	}
	if len(ids) == 0 {
		response.Error(c, http.StatusBadRequest, "reminder ids required", nil)
		return
	}

	completed := true
	if req.Completed != nil {
		completed = *req.Completed
	}

	if err := h.student.BatchSetReminderCompletion(c.Request.Context(), accountID, ids, completed); err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		case errors.Is(err, service.ErrStudentReminderInvalid):
			response.Error(c, http.StatusBadRequest, "invalid reminder ids", nil)
		case errors.Is(err, service.ErrStudentReminderNotFound):
			response.Error(c, http.StatusNotFound, "reminders not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to update reminders", err.Error())
		}
		return
	}

	response.Success(c, http.StatusNoContent, nil)
}

func (h *Handler) UpdateAllStudentRemindersCompletion(c *gin.Context) {
	var req reminderCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	completed := true
	if req.Completed != nil {
		completed = *req.Completed
	}

	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	if err := h.student.SetAllRemindersCompletion(c.Request.Context(), accountID, completed); err != nil {
		switch {
		case errors.Is(err, service.ErrStudentProfileNotFound):
			response.Error(c, http.StatusNotFound, "student profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to update reminders", err.Error())
		}
		return
	}

	response.Success(c, http.StatusNoContent, nil)
}

func (h *Handler) ListTeacherSchedule(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	start, end, err := parseScheduleRange(c.Query("from"), c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid time range", err.Error())
		return
	}

	items, err := h.teacher.ListSchedule(c.Request.Context(), accountID, start, end)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch schedule", err.Error())
		}
		return
	}

	sessions := make([]gin.H, 0, len(items))
	for _, item := range items {
		sessions = append(sessions, gin.H{
			"session_id":  item.SessionID,
			"course_id":   item.CourseID,
			"course_name": item.CourseName,
			"class_id":    item.ClassID,
			"class_name":  item.ClassName,
			"starts_at":   item.StartsAt,
			"ends_at":     item.EndsAt,
			"day":         item.Day,
			"slot_id":     item.SlotID,
			"slot_name":   item.SlotName,
			"location":    item.Location,
			"source":      item.Source,
		})
	}

	response.Success(c, http.StatusOK, gin.H{"sessions": sessions})
}

func (h *Handler) ListTeacherAssignments(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	limit := parseLimit(c.DefaultQuery("limit", "20"), 20, 200)
	classID := c.Query("class_id")
	types := parseAssignmentTypes(c.Query("types"))

	items, err := h.teacher.ListAssignments(c.Request.Context(), accountID, limit, classID, types)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch assignments", err.Error())
		}
		return
	}

	assignments := h.buildTeacherAssignmentPayloads(items)

	response.Success(c, http.StatusOK, gin.H{"assignments": assignments})
}

func (h *Handler) ListTeacherExams(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	limit := parseLimit(c.DefaultQuery("limit", "20"), 20, 200)
	classID := c.Query("class_id")

	items, err := h.teacher.ListExams(c.Request.Context(), accountID, limit, classID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch exams", err.Error())
		}
		return
	}

	response.Success(c, http.StatusOK, gin.H{"exams": h.buildTeacherAssignmentPayloads(items)})
}

func (h *Handler) buildTeacherAssignmentPayloads(items []service.TeacherAssignmentItem) []gin.H {
	assignments := make([]gin.H, 0, len(items))
	for _, item := range items {
		payload := gin.H{
			"id":                  item.ID,
			"title":               item.Title,
			"description":         item.Description,
			"course_id":           item.CourseID,
			"course_name":         item.CourseName,
			"class_id":            item.ClassID,
			"class_name":          item.ClassName,
			"type":                string(item.Type),
			"allow_resubmit":      item.AllowResubmit,
			"submission_count":    item.SubmissionCount,
			"submitted_count":     item.SubmittedCount,
			"graded_count":        item.GradedCount,
			"pending_grade_count": item.PendingGradeCount,
			"missing_count":       item.MissingCount,
			"score_distribution": gin.H{
				"below_60":      item.ScoreDistribution.Below60,
				"between_60_70": item.ScoreDistribution.Between60And70,
				"between_70_80": item.ScoreDistribution.Between70And80,
				"between_80_90": item.ScoreDistribution.Between80And90,
				"above_90":      item.ScoreDistribution.Above90,
			},
		}
		if item.StartAt != nil {
			payload["start_at"] = item.StartAt
		}
		if item.DueAt != nil {
			payload["due_at"] = item.DueAt
		}
		if item.LatestSubmissionAt != nil {
			payload["latest_submission_at"] = item.LatestSubmissionAt
		}
		if item.ScoreAverage != nil {
			payload["score_average"] = item.ScoreAverage
		}
		if item.ScoreMax != nil {
			payload["score_max"] = item.ScoreMax
		}
		if item.ScoreMin != nil {
			payload["score_min"] = item.ScoreMin
		}
		assignments = append(assignments, payload)
	}
	return assignments
}

func (h *Handler) GetTeacherAssignment(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}
	assignmentID := c.Param("id")
	if assignmentID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment id", nil)
		return
	}

	includeAnswers := parseBool(c.DefaultQuery("include_answers", "false"), false)

	detail, err := h.teacher.GetAssignmentDetail(c.Request.Context(), accountID, assignmentID, includeAnswers)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		case errors.Is(err, service.ErrTeacherAssignmentForbidden):
			response.Error(c, http.StatusForbidden, "assignment not accessible", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch assignment", err.Error())
		}
		return
	}

	assignment := gin.H{
		"id":             detail.ID,
		"title":          detail.Title,
		"description":    detail.Description,
		"course_id":      detail.CourseID,
		"course_name":    detail.CourseName,
		"class_id":       detail.ClassID,
		"class_name":     detail.ClassName,
		"type":           string(detail.Type),
		"allow_resubmit": detail.AllowResubmit,
		"max_score":      detail.MaxScore,
	}
	if detail.StartAt != nil {
		assignment["start_at"] = detail.StartAt
	}
	if detail.DueAt != nil {
		assignment["due_at"] = detail.DueAt
	}

	questions := make([]gin.H, 0, len(detail.Questions))
	for _, q := range detail.Questions {
		entry := gin.H{
			"id":          q.ID,
			"type":        string(q.Type),
			"prompt":      q.Prompt,
			"score":       q.Score,
			"order_index": q.OrderIndex,
		}
		if includeAnswers {
			if q.Options != "" {
				entry["options"] = q.Options
			}
			if q.Answer != "" {
				entry["answer"] = q.Answer
			}
		}
		questions = append(questions, entry)
	}

	stats := gin.H{
		"submission_count":    detail.Stats.SubmissionCount,
		"submitted_count":     detail.Stats.SubmittedCount,
		"graded_count":        detail.Stats.GradedCount,
		"pending_grade_count": detail.Stats.PendingGradeCount,
		"missing_count":       detail.Stats.MissingCount,
		"class_student_count": detail.Stats.ClassStudentCount,
	}
	if detail.Stats.ScoreAverage != nil {
		stats["score_average"] = detail.Stats.ScoreAverage
	}
	if detail.Stats.ScoreMax != nil {
		stats["score_max"] = detail.Stats.ScoreMax
	}
	if detail.Stats.ScoreMin != nil {
		stats["score_min"] = detail.Stats.ScoreMin
	}
	if detail.Stats.LatestSubmissionAt != nil {
		stats["latest_submission_at"] = detail.Stats.LatestSubmissionAt
	}

	response.Success(c, http.StatusOK, gin.H{
		"assignment": assignment,
		"questions":  questions,
		"stats":      stats,
	})
}

func (h *Handler) ExportTeacherAssignmentGrades(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}
	assignmentID := c.Param("id")
	if assignmentID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment id", nil)
		return
	}

	export, err := h.teacher.ExportAssignmentGrades(c.Request.Context(), accountID, assignmentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		case errors.Is(err, service.ErrTeacherAssignmentForbidden):
			response.Error(c, http.StatusForbidden, "assignment not accessible", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to export grades", err.Error())
		}
		return
	}

	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	headers := []string{"student_id", "student_number", "student_name", "status", "score", "submitted_at"}
	if err := writer.Write(headers); err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to export grades", err.Error())
		return
	}

	for _, row := range export.Rows {
		scoreValue := ""
		if row.Score != nil {
			scoreValue = strconv.FormatFloat(*row.Score, 'f', 2, 64)
		}
		submittedAt := ""
		if row.SubmittedAt != nil {
			submittedAt = row.SubmittedAt.UTC().Format(time.RFC3339)
		}
		record := []string{
			row.StudentID,
			row.StudentNumber,
			row.StudentName,
			row.Status,
			scoreValue,
			submittedAt,
		}
		if err := writer.Write(record); err != nil {
			response.Error(c, http.StatusInternalServerError, "unable to export grades", err.Error())
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to export grades", err.Error())
		return
	}

	filename := fmt.Sprintf("%s-%s-grades.csv", safeFilenameComponent(export.ClassName), safeFilenameComponent(export.AssignmentTitle))
	if filename == "-grades.csv" {
		filename = "assignment-grades.csv"
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func (h *Handler) ListTeacherAgenda(c *gin.Context) {
	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	start, end, err := parseScheduleRange(c.Query("from"), c.Query("to"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid time range", err.Error())
		return
	}
	includeAssignments := parseBool(c.DefaultQuery("include_assignments", "true"), true)

	items, err := h.teacher.ListAgenda(c.Request.Context(), accountID, start, end, includeAssignments)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeacherProfileNotFound):
			response.Error(c, http.StatusNotFound, "teacher profile not found", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to fetch agenda", err.Error())
		}
		return
	}

	agenda := make([]gin.H, 0, len(items))
	for _, item := range items {
		entry := gin.H{
			"id":          item.ID,
			"kind":        item.Kind,
			"title":       item.Title,
			"description": item.Description,
			"course_id":   item.CourseID,
			"course_name": item.CourseName,
			"class_id":    item.ClassID,
			"class_name":  item.ClassName,
			"starts_at":   item.StartsAt,
			"day":         item.Day,
			"slot_id":     item.SlotID,
			"slot_name":   item.SlotName,
			"location":    item.Location,
			"source":      item.Source,
		}
		if item.EndsAt != nil {
			entry["ends_at"] = item.EndsAt
		}
		agenda = append(agenda, entry)
	}

	response.Success(c, http.StatusOK, gin.H{"agenda": agenda})
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

	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	teacherID, err := h.teacher.GetTeacherID(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, service.ErrTeacherProfileNotFound) {
			response.Error(c, http.StatusForbidden, "teacher profile required", nil)
		} else {
			response.Error(c, http.StatusInternalServerError, "unable to resolve teacher profile", err.Error())
		}
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

func parseLimit(raw string, defVal, maxVal int) int {
	val, err := strconv.Atoi(raw)
	if err != nil || val <= 0 {
		return defVal
	}
	if val > maxVal {
		return maxVal
	}
	return val
}

func parseBool(raw string, defaultVal bool) bool {
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultVal
	}
	return v
}

func parseScheduleRange(fromRaw, toRaw string) (time.Time, time.Time, error) {
	loc := time.Now().Location()
	var start time.Time
	var err error
	if fromRaw == "" {
		start = time.Now().Truncate(24 * time.Hour)
	} else {
		start, err = time.Parse(time.RFC3339, fromRaw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)

	var end time.Time
	if toRaw == "" {
		end = start.AddDate(0, 0, 7)
	} else {
		end, err = time.Parse(time.RFC3339, toRaw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("end must be after start")
	}

	return start, end, nil
}

func parseAssignmentTypes(raw string) []domain.AssignmentType {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]domain.AssignmentType, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		switch strings.ToLower(part) {
		case string(domain.AssignmentHomework):
			result = append(result, domain.AssignmentHomework)
		case string(domain.AssignmentExam):
			result = append(result, domain.AssignmentExam)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func studentReminderPayload(item domain.StudentReminder) gin.H {
	payload := gin.H{
		"id":           item.ID,
		"title":        item.Title,
		"description":  item.Description,
		"time_label":   item.TimeLabel,
		"priority":     item.Priority,
		"icon":         item.Icon,
		"route":        item.Route,
		"is_completed": item.CompletedAt != nil,
	}
	if item.CompletedAt != nil {
		payload["completed_at"] = item.CompletedAt
	}
	return payload
}

func parseReminderPriority(raw string) domain.StudentReminderPriority {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(domain.StudentReminderPriorityHigh):
		return domain.StudentReminderPriorityHigh
	default:
		return domain.StudentReminderPriorityNormal
	}
}

func safeFilenameComponent(input string) string {
	lowered := strings.ToLower(strings.TrimSpace(input))
	if lowered == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range lowered {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		case unicode.IsSpace(r):
			builder.WriteRune('-')
		default:
			builder.WriteRune('-')
		}
	}
	return strings.Trim(builder.String(), "-")
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

	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	teacherID, err := h.teacher.GetTeacherID(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, service.ErrTeacherProfileNotFound) {
			response.Error(c, http.StatusForbidden, "teacher profile required", nil)
		} else {
			response.Error(c, http.StatusInternalServerError, "unable to resolve teacher profile", err.Error())
		}
		return
	}

	input := service.GradeSubmissionInput{
		AssignmentID: assignmentID,
		SubmissionID: submissionID,
		AccountID:    accountID,
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

type returnSubmissionRequest struct {
	Comment string `json:"comment"`
}

func (h *Handler) ReturnSubmission(c *gin.Context) {
	assignmentID := strings.TrimSpace(c.Param("id"))
	submissionID := strings.TrimSpace(c.Param("submissionID"))
	if assignmentID == "" || submissionID == "" {
		response.Error(c, http.StatusBadRequest, "missing assignment or submission id", nil)
		return
	}

	var req returnSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	accountID := getAccountID(c)
	if accountID == "" {
		response.Error(c, http.StatusUnauthorized, "missing account context", nil)
		return
	}

	teacherID, err := h.teacher.GetTeacherID(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, service.ErrTeacherProfileNotFound) {
			response.Error(c, http.StatusForbidden, "teacher profile required", nil)
		} else {
			response.Error(c, http.StatusInternalServerError, "unable to resolve teacher profile", err.Error())
		}
		return
	}

	detail, comments, err := h.assignments.ReturnSubmission(c.Request.Context(), teacherID, service.ReturnSubmissionInput{
		AssignmentID: assignmentID,
		SubmissionID: submissionID,
		AccountID:    accountID,
		Comment:      req.Comment,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssignmentNotFound):
			response.Error(c, http.StatusNotFound, "assignment not found", nil)
		case errors.Is(err, service.ErrSubmissionNotFound):
			response.Error(c, http.StatusNotFound, "submission not found", nil)
		case errors.Is(err, service.ErrSubmissionForbidden):
			response.Error(c, http.StatusForbidden, "submission forbidden", nil)
		default:
			response.Error(c, http.StatusInternalServerError, "unable to return submission", err.Error())
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
		name := ""
		if acc, ok := summary.MemberProfiles[member.AccountID]; ok {
			name = acc.DisplayName
		}
		members = append(members, gin.H{
			"id":              member.ID,
			"conversation_id": member.ConversationID,
			"account_id":      member.AccountID,
			"account_name":    name,
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

func aiChatSessionPayload(session domain.AIChatSession) gin.H {
	return gin.H{
		"id":              session.ID,
		"school_id":       session.SchoolID,
		"account_id":      session.AccountID,
		"role":            string(session.Role),
		"title":           session.Title,
		"last_message_at": session.LastMessageAt,
		"message_count":   session.MessageCount,
		"token_count":     session.TokenCount,
		"created_at":      session.CreatedAt,
		"updated_at":      session.UpdatedAt,
		"closed_at":       session.ClosedAt,
	}
}

func aiChatMessagePayload(msg domain.AIChatMessage) gin.H {
	return gin.H{
		"id":            msg.ID,
		"session_id":    msg.SessionID,
		"sender":        msg.Sender,
		"content":       msg.Content,
		"prompt_tokens": msg.PromptTokens,
		"result_tokens": msg.ResultTokens,
		"latency_ms":    msg.LatencyMS,
		"created_at":    msg.CreatedAt,
	}
}

func aiUsageSummaryPayload(summary service.AIChatUsageSummary) gin.H {
	return gin.H{
		"account_id":               summary.AccountID,
		"account_name":             summary.AccountName,
		"role":                     string(summary.Role),
		"school_id":                summary.SchoolID,
		"since":                    summary.Since,
		"user_messages":            summary.UserMessages,
		"assistant_messages":       summary.AssistantMessages,
		"total_messages":           summary.TotalMessages,
		"prompt_tokens":            summary.PromptTokens,
		"result_tokens":            summary.ResultTokens,
		"total_tokens":             summary.TotalTokens,
		"max_daily_requests":       summary.MaxDailyRequests,
		"remaining_daily_requests": summary.RemainingDailyRequests,
	}
}

func aiUsageAggregatePayload(aggregate service.AIChatUsageAggregate, breakdown []service.AIChatUsageRoleSummary) gin.H {
	payload := gin.H{
		"school_id":          aggregate.SchoolID,
		"role_filter":        string(aggregate.RoleFilter),
		"since":              aggregate.Since,
		"active_accounts":    aggregate.ActiveAccounts,
		"user_messages":      aggregate.UserMessages,
		"assistant_messages": aggregate.AssistantMessages,
		"total_messages":     aggregate.TotalMessages,
		"prompt_tokens":      aggregate.PromptTokens,
		"result_tokens":      aggregate.ResultTokens,
		"total_tokens":       aggregate.TotalTokens,
		"max_daily_requests": aggregate.MaxDailyRequests,
	}

	if len(breakdown) > 0 {
		summaries := make([]service.AIChatUsageRoleSummary, len(breakdown))
		copy(summaries, breakdown)
		rolePriority := map[domain.Role]int{
			domain.RoleTeacher: 0,
			domain.RoleStudent: 1,
			domain.RoleAdmin:   2,
		}
		sort.SliceStable(summaries, func(i, j int) bool {
			pi, okI := rolePriority[summaries[i].Role]
			pj, okJ := rolePriority[summaries[j].Role]
			switch {
			case okI && okJ:
				if pi == pj {
					return string(summaries[i].Role) < string(summaries[j].Role)
				}
				return pi < pj
			case okI:
				return true
			case okJ:
				return false
			default:
				return string(summaries[i].Role) < string(summaries[j].Role)
			}
		})

		items := make([]gin.H, len(summaries))
		for i, summary := range summaries {
			items[i] = aiUsageRoleSummaryPayload(summary)
		}
		payload["by_role"] = items
	}

	return payload
}

func aiUsageRoleSummaryPayload(summary service.AIChatUsageRoleSummary) gin.H {
	return gin.H{
		"role":               string(summary.Role),
		"account_count":      summary.AccountCount,
		"user_messages":      summary.UserMessages,
		"assistant_messages": summary.AssistantMessages,
		"total_messages":     summary.TotalMessages,
		"prompt_tokens":      summary.PromptTokens,
		"result_tokens":      summary.ResultTokens,
		"total_tokens":       summary.TotalTokens,
		"message_share":      summary.MessageShare,
		"token_share":        summary.TokenShare,
		"avg_messages":       summary.AverageMessages,
		"avg_tokens":         summary.AverageTokens,
	}
}

func aiUsageTimelinePayload(points []service.AIChatUsageTimelinePoint) []gin.H {
	items := make([]gin.H, 0, len(points))
	for _, point := range points {
		items = append(items, gin.H{
			"date":               point.Date,
			"account_count":      point.AccountCount,
			"user_messages":      point.UserMessages,
			"assistant_messages": point.AssistantMessages,
			"total_messages":     point.TotalMessages,
			"prompt_tokens":      point.PromptTokens,
			"result_tokens":      point.ResultTokens,
			"total_tokens":       point.TotalTokens,
			"avg_messages":       point.AverageMessages,
			"avg_tokens":         point.AverageTokens,
		})
	}
	return items
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

// ListSchools returns all schools.
func (h *Handler) ListSchools(c *gin.Context) {
	schools, err := h.school.ListSchools(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "unable to list schools", err.Error())
		return
	}

	items := make([]gin.H, 0, len(schools))
	for _, s := range schools {
		items = append(items, gin.H{
			"id":   s.ID,
			"name": s.Name,
		})
	}

	response.Success(c, http.StatusOK, gin.H{"schools": items})
}
