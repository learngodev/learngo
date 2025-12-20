package repository

import (
	"context"
	"errors"
	"time"

	"learn-go/internal/domain"
)

// ErrNotFound indicates repository query returned no rows.
var ErrNotFound = errors.New("repository: not found")

// AccountRepository defines persistence for accounts.
type AccountRepository interface {
	Create(ctx context.Context, account *domain.Account) error
	FindByIdentifier(ctx context.Context, schoolID, identifier string) (*domain.Account, error)
	FindByID(ctx context.Context, id string) (*domain.Account, error)
	ListByIDs(ctx context.Context, ids []string) ([]domain.Account, error)
	ListByRole(
		ctx context.Context,
		schoolID string,
		role domain.Role,
		status domain.AccountStatus,
		departmentID string,
		classID string,
		courseID string,
		onlyClassless bool,
		onlyDepartmentless bool,
		page int,
		size int,
		query string,
	) ([]domain.Account, int64, error)
	UpdateStatus(ctx context.Context, accountID, schoolID string, status domain.AccountStatus) error
	UpdatePasswordHash(ctx context.Context, accountID string, passwordHash string) error
	Delete(ctx context.Context, accountID, schoolID string) error
}

// SchoolRepository handles school persistence.
type SchoolRepository interface {
	List(ctx context.Context) ([]domain.School, error)
}

// PasswordResetTokenRepository manages password reset tokens.
type PasswordResetTokenRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	FindByTokenHash(ctx context.Context, hash string) (*domain.PasswordResetToken, error)
	Consume(ctx context.Context, id string, consumedAt time.Time) error
	DeleteByAccount(ctx context.Context, accountID string) error
}

// TeacherRepository handles teacher profile persistence.
type TeacherRepository interface {
	Create(ctx context.Context, teacher *domain.Teacher) error
	GetByNumber(ctx context.Context, schoolID, number string) (*domain.Teacher, error)
	GetByID(ctx context.Context, id string) (*domain.Teacher, error)
	GetByAccountID(ctx context.Context, accountID string) (*domain.Teacher, error)
}

// StudentRepository handles student profile persistence.
type StudentRepository interface {
	Create(ctx context.Context, student *domain.Student) error
	GetByNumber(ctx context.Context, schoolID, number string) (*domain.Student, error)
	GetByID(ctx context.Context, id string) (*domain.Student, error)
	GetByAccountID(ctx context.Context, accountID string) (*domain.Student, error)
	ListByIDs(ctx context.Context, ids []string) ([]domain.Student, error)
	ListByClassID(ctx context.Context, classID string) ([]domain.Student, error)
	ListByDepartmentID(ctx context.Context, departmentID string) ([]domain.Student, error)
	CountByClassIDs(ctx context.Context, classIDs []string) (map[string]int64, error)
	UpdateClassID(ctx context.Context, studentID string, classID string) error
}

// StudentReminderRepository persists custom reminders created by students.
type StudentReminderRepository interface {
	ListByStudent(ctx context.Context, studentID string) ([]domain.StudentReminder, error)
	GetByID(ctx context.Context, id string) (*domain.StudentReminder, error)
	Create(ctx context.Context, reminder *domain.StudentReminder) error
	UpdateFields(ctx context.Context, id string, studentID string, updates map[string]any) (*domain.StudentReminder, error)
	Delete(ctx context.Context, id string, studentID string) error
	MarkAllCompleted(ctx context.Context, studentID string, completed bool, timestamp *time.Time) error
	SetCompletion(ctx context.Context, id string, studentID string, completed bool, timestamp *time.Time) (*domain.StudentReminder, error)
	MarkBatchCompleted(ctx context.Context, studentID string, ids []string, completed bool, timestamp *time.Time) error
}

// CourseRepository retrieves course metadata.
type CourseRepository interface {
	Create(ctx context.Context, course *domain.Course) error
	List(ctx context.Context, schoolID string, page, size int) ([]domain.Course, int64, error)
	GetByID(ctx context.Context, id string) (*domain.Course, error)
	Update(ctx context.Context, course *domain.Course) error
	Delete(ctx context.Context, id string) error
	ListByIDs(ctx context.Context, ids []string) ([]domain.Course, error)
	ListAssignments(ctx context.Context, schoolID string, courseID, departmentID, classID string, onlyAssigned bool, page, size int) ([]domain.CourseAssignmentInfo, int64, error)
	ListWithFilters(ctx context.Context, schoolID string, departmentID, classID string, page, size int) ([]domain.Course, int64, error)
	ListAssignmentsByCourseIDs(ctx context.Context, courseIDs []string) ([]domain.CourseAssignmentInfo, error)
}

// CourseStudentRepository manages student enrollments in courses.
type CourseStudentRepository interface {
	BatchCreate(ctx context.Context, enrollments []domain.CourseStudent) error
	ListByCourseID(ctx context.Context, courseID string) ([]domain.CourseStudent, error)
	DeleteByCourseAndStudent(ctx context.Context, courseID string, studentIDs []string) error
}

// CourseScheduleRepository manages recurring schedule rules.
type CourseScheduleRepository interface {
	Create(ctx context.Context, schedule *domain.CourseSchedule) error
	Update(ctx context.Context, schedule *domain.CourseSchedule) error
	Delete(ctx context.Context, id string) error
	ListByCourse(ctx context.Context, courseID string) ([]domain.CourseSchedule, error)
	ListBySchool(ctx context.Context, schoolID string) ([]domain.CourseSchedule, error)
	ListDetailsBySchool(ctx context.Context, schoolID string, courseID string) ([]domain.CourseScheduleDetail, error)
	GetStats(ctx context.Context, schoolID string) (*domain.ScheduleStats, error)
	ListByClassroom(ctx context.Context, classroomID string) ([]domain.CourseSchedule, error)
	ListByTeacher(ctx context.Context, teacherID string) ([]domain.CourseSchedule, error)
	ListByClass(ctx context.Context, classID string) ([]domain.CourseSchedule, error)
}

// ClassroomRepository manages classroom persistence.
type ClassroomRepository interface {
	Create(ctx context.Context, classroom *domain.Classroom) error
	GetByID(ctx context.Context, id string) (*domain.Classroom, error)
	List(ctx context.Context, schoolID string, page, size int) ([]domain.Classroom, int64, error)
	Update(ctx context.Context, classroom *domain.Classroom) error
	Delete(ctx context.Context, id string) error
}

// CourseSessionRepository retrieves scheduled lessons.
type CourseSessionRepository interface {
	Create(ctx context.Context, session *domain.CourseSession) error
	Update(ctx context.Context, session *domain.CourseSession) error
	GetByID(ctx context.Context, id string) (*domain.CourseSession, error)
	ListByClassBetween(ctx context.Context, classID string, start, end time.Time) ([]domain.CourseSession, error)
	ListByTeacherBetween(ctx context.Context, teacherID string, start, end time.Time) ([]domain.CourseSession, error)
}

// TeacherStudentRepository manages relationships between teachers and students.
type TeacherStudentRepository interface {
	BindTeachers(ctx context.Context, studentID string, teacherIDs []string) error
}

// DepartmentRepository handles departments.
type DepartmentRepository interface {
	Create(ctx context.Context, department *domain.Department) error
	List(ctx context.Context, schoolID string) ([]domain.Department, error)
	GetByID(ctx context.Context, id string) (*domain.Department, error)
	UpdateName(ctx context.Context, id, schoolID, name string) error
	Delete(ctx context.Context, id, schoolID string) error
}

// ClassRepository handles classes.
type ClassRepository interface {
	Create(ctx context.Context, class *domain.Class) error
	ListByDepartment(ctx context.Context, schoolID, departmentID string) ([]domain.Class, error)
	GetByID(ctx context.Context, id string) (*domain.Class, error)
	ListByIDs(ctx context.Context, ids []string) ([]domain.Class, error)
	UpdateName(ctx context.Context, id, schoolID, name string) error
	Delete(ctx context.Context, id, schoolID string) error
}

// AssignmentRepository handles assignments.
type AssignmentRepository interface {
	Create(ctx context.Context, assignment *domain.Assignment, questions []domain.AssignmentQuestion) error
	Get(ctx context.Context, id string) (*domain.Assignment, []domain.AssignmentQuestion, error)
	ListByClass(ctx context.Context, classID string, limit int, types []domain.AssignmentType) ([]domain.Assignment, error)
	ListDueBetween(ctx context.Context, classID string, start, end time.Time, types []domain.AssignmentType) ([]domain.Assignment, error)
	ListByTeacher(ctx context.Context, teacherID string, limit int, classID string, types []domain.AssignmentType) ([]domain.Assignment, error)
	ListDueBetweenByTeacher(ctx context.Context, teacherID string, start, end time.Time, types []domain.AssignmentType) ([]domain.Assignment, error)
	Search(ctx context.Context, teacherID string, query string) ([]domain.Assignment, error)
	Update(ctx context.Context, assignment *domain.Assignment) error
}

// SubmissionRepository handles student submissions.
type SubmissionRepository interface {
	CreateOrUpdate(ctx context.Context, submission *domain.AssignmentSubmission, items []domain.SubmissionItem) error
	ListByAssignment(ctx context.Context, assignmentID string) ([]domain.AssignmentSubmission, error)
	ListItemsBySubmissionIDs(ctx context.Context, submissionIDs []string) ([]domain.SubmissionItem, error)
	GetByAssignmentAndStudent(ctx context.Context, assignmentID, studentID string) (*domain.AssignmentSubmission, []domain.SubmissionItem, error)
	GetByID(ctx context.Context, submissionID string) (*domain.AssignmentSubmission, []domain.SubmissionItem, error)
	UpdateGrades(ctx context.Context, submission *domain.AssignmentSubmission, items []domain.SubmissionItem) error
	ListByStudentAndAssignments(ctx context.Context, studentID string, assignmentIDs []string) ([]domain.AssignmentSubmission, error)
	StatsByAssignments(ctx context.Context, assignmentIDs []string) ([]AssignmentSubmissionStats, error)
}

// AssignmentSubmissionStats holds aggregated submission data per assignment.
type AssignmentSubmissionStats struct {
	AssignmentID      string
	Total             int64
	Submitted         int64
	Graded            int64
	LatestSubmittedAt *time.Time
	AverageScore      *float64
	MaxScore          *float64
	MinScore          *float64
	ScoreDistribution AssignmentScoreDistribution
}

// AssignmentScoreDistribution holds counts for score buckets.
type AssignmentScoreDistribution struct {
	Below60        int64
	Between60And70 int64
	Between70And80 int64
	Between80And90 int64
	Above90        int64
}

// SubmissionCommentRepository handles submission review comments.
type SubmissionCommentRepository interface {
	Create(ctx context.Context, comment *domain.SubmissionComment) error
	ListBySubmission(ctx context.Context, submissionID string) ([]domain.SubmissionComment, error)
}

// NoteRepository handles note persistence.
type NoteRepository interface {
	Create(ctx context.Context, note *domain.Note) error
	Update(ctx context.Context, note *domain.Note) error
	FindByID(ctx context.Context, id string) (*domain.Note, error)
	ListByOwner(ctx context.Context, ownerID string, includeDeleted bool, status string) ([]domain.Note, error)
	ListPublishedBySchool(ctx context.Context, schoolID string) ([]domain.Note, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
}

// NoteCommentRepository handles note comment persistence.
type NoteCommentRepository interface {
	Create(ctx context.Context, comment *domain.NoteComment) error
	ListByNote(ctx context.Context, noteID string) ([]domain.NoteComment, error)
}

// ConversationRepository handles conversation persistence and membership.
type ConversationRepository interface {
	Create(ctx context.Context, conversation *domain.Conversation, members []domain.ConversationMember) error
	GetByID(ctx context.Context, id string) (*domain.Conversation, error)
	ListByAccount(ctx context.Context, accountID string) ([]domain.Conversation, error)
	GetMembers(ctx context.Context, conversationID string) ([]domain.ConversationMember, error)
	IsMember(ctx context.Context, conversationID, accountID string) (bool, error)
	FindDirectBetween(ctx context.Context, schoolID string, participantIDs [2]string) (*domain.Conversation, error)
	UpdateTimestamp(ctx context.Context, conversationID string, ts time.Time) error
}

// MessageRepository handles chat messages.
type MessageRepository interface {
	Create(ctx context.Context, message *domain.Message) error
	ListByConversation(ctx context.Context, conversationID string, limit int, beforeID string) ([]domain.Message, error)
	GetLastByConversation(ctx context.Context, conversationID string) (*domain.Message, error)
	GetByID(ctx context.Context, id string) (*domain.Message, error)
}

// MessageReceiptRepository records read state for messages.
type MessageReceiptRepository interface {
	CreateBatch(ctx context.Context, receipts []domain.MessageReceipt) error
	CountUnread(ctx context.Context, accountID, conversationID string) (int64, error)
	MarkReadUpTo(ctx context.Context, accountID, conversationID string, ts time.Time) error
}

// OssCredentialRepository manages OSS credential persistence.
type OssCredentialRepository interface {
	Create(ctx context.Context, credential *domain.OssCredential) error
	List(ctx context.Context, schoolID string) ([]domain.OssCredential, error)
	GetByID(ctx context.Context, credentialID, schoolID string) (*domain.OssCredential, error)
	Update(ctx context.Context, credentialID, schoolID string, updates map[string]any) (*domain.OssCredential, error)
	SetPrimary(ctx context.Context, credentialID, schoolID string) (*domain.OssCredential, error)
	Delete(ctx context.Context, credentialID, schoolID string) error
}

// OssPolicyRepository manages OSS policy persistence.
type OssPolicyRepository interface {
	Create(ctx context.Context, policy *domain.OssPolicy) error
	List(ctx context.Context, schoolID string) ([]domain.OssPolicy, error)
	GetByID(ctx context.Context, policyID, schoolID string) (*domain.OssPolicy, error)
	UpdateStatus(ctx context.Context, policyID, schoolID string, status domain.OssPolicyStatus) (*domain.OssPolicy, error)
	Delete(ctx context.Context, policyID, schoolID string) error
}

// OssAuditRepository records OSS configuration changes.
type OssAuditRepository interface {
	Create(ctx context.Context, log *domain.OssAuditLog) error
	ListRecent(ctx context.Context, schoolID string, limit int) ([]domain.OssAuditLog, error)
}

// AIAgentSettingRepository manages AI assistant configuration records.
type AIAgentSettingRepository interface {
	GetBySchoolID(ctx context.Context, schoolID string) (*domain.AIAgentSetting, error)
	Upsert(ctx context.Context, setting *domain.AIAgentSetting) error
}

// AIAgentSettingAuditRepository stores administrative audits for AI settings.
type AIAgentSettingAuditRepository interface {
	Create(ctx context.Context, entry *domain.AIAgentSettingAudit) error
	ListRecent(ctx context.Context, schoolID string, limit int) ([]domain.AIAgentSettingAudit, error)
}

// AIChatSessionRepository persists AI assistant session metadata.
type AIChatSessionRepository interface {
	Create(ctx context.Context, session *domain.AIChatSession) error
	GetByID(ctx context.Context, sessionID string) (*domain.AIChatSession, error)
	ListByAccount(ctx context.Context, accountID string, limit int) ([]domain.AIChatSession, error)
	UpdateFields(ctx context.Context, sessionID string, updates map[string]any) error
	Delete(ctx context.Context, sessionID string) error
}

// AIChatMessageRepository persists AI assistant message transcripts.
type AIChatMessageRepository interface {
	Create(ctx context.Context, message *domain.AIChatMessage) error
	ListBySession(ctx context.Context, sessionID string, limit int, before time.Time) ([]domain.AIChatMessage, error)
	CountUserMessagesSince(ctx context.Context, accountID string, since time.Time) (int64, error)
	UsageStatsByAccountSince(ctx context.Context, accountID string, since time.Time) (AIChatUsageStats, error)
	UsageStatsBySchoolSince(ctx context.Context, schoolID string, since time.Time, role domain.Role, limit int, offset int, sort AIChatUsageSort) ([]AIChatAccountUsage, error)
	UsageTotalsBySchoolSince(ctx context.Context, schoolID string, since time.Time, role domain.Role) (AIChatUsageTotals, error)
	UsageByRoleSince(ctx context.Context, schoolID string, since time.Time) (map[domain.Role]AIChatUsageTotals, error)
	UsageTimelineBySchool(ctx context.Context, schoolID string, start time.Time, end time.Time, role domain.Role) ([]AIChatUsageTimelinePoint, error)
}

// SortDirection enumerates basic ordering options.
type SortDirection string

const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)

// AIChatUsageSortField enumerates sortable usage columns.
type AIChatUsageSortField string

const (
	AIChatUsageSortUserMessages  AIChatUsageSortField = "user_messages"
	AIChatUsageSortTotalMessages AIChatUsageSortField = "total_messages"
	AIChatUsageSortTotalTokens   AIChatUsageSortField = "total_tokens"
)

// AIChatUsageSort configures ordering for usage stats queries.
type AIChatUsageSort struct {
	Field     AIChatUsageSortField
	Direction SortDirection
}

// AIChatUsageStats aggregates AI usage activity.
type AIChatUsageStats struct {
	UserMessages      int64
	AssistantMessages int64
	PromptTokens      int64
	ResultTokens      int64
}

// AIChatAccountUsage aggregates usage information for a single account.
type AIChatAccountUsage struct {
	AccountID string
	AIChatUsageStats
}

// AIChatUsageTotals aggregates usage across a school.
type AIChatUsageTotals struct {
	AccountCount int64
	AIChatUsageStats
}

// AIChatUsageTimelinePoint aggregates usage within a fixed time bucket.
type AIChatUsageTimelinePoint struct {
	Bucket       time.Time
	AccountCount int64
	AIChatUsageStats
}

// SystemSwitchRepository persists admin feature toggles.
type SystemSwitchRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []domain.SystemSwitch) error
	List(ctx context.Context, schoolID string) ([]domain.SystemSwitch, error)
	Get(ctx context.Context, schoolID, id string) (*domain.SystemSwitch, error)
	UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*domain.SystemSwitch, error)
}

// SystemParameterRepository persists configurable parameters.
type SystemParameterRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []domain.SystemParameter) error
	List(ctx context.Context, schoolID string) ([]domain.SystemParameter, error)
	Get(ctx context.Context, schoolID, id string) (*domain.SystemParameter, error)
	UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*domain.SystemParameter, error)
}

// SystemBroadcastRepository persists announcement records.
type SystemBroadcastRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []domain.SystemBroadcast) error
	List(ctx context.Context, schoolID string) ([]domain.SystemBroadcast, error)
	Get(ctx context.Context, schoolID, id string) (*domain.SystemBroadcast, error)
	UpdateFields(ctx context.Context, schoolID, id string, updates map[string]any) (*domain.SystemBroadcast, error)
}

// SystemAuditRepository stores audit trail for system settings.
type SystemAuditRepository interface {
	EnsureDefaults(ctx context.Context, schoolID string, defaults []domain.SystemAuditLog) error
	Create(ctx context.Context, entry *domain.SystemAuditLog) error
	List(ctx context.Context, schoolID string, limit int) ([]domain.SystemAuditLog, error)
}

// TimeSlotRepository manages class time slots.
type TimeSlotRepository interface {
	Create(ctx context.Context, slot *domain.TimeSlot) error
	Update(ctx context.Context, slot *domain.TimeSlot) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, schoolID string) ([]domain.TimeSlot, error)
	FindByID(ctx context.Context, id string) (*domain.TimeSlot, error)
}

// CourseSlotRepository manages course slots (alias for TimeSlotRepository for now).
type CourseSlotRepository interface {
	ListByIDs(ctx context.Context, ids []string) ([]domain.CourseSlot, error)
}
