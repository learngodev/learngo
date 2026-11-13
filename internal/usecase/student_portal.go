package usecase

import (
	"context"
	"errors"
	"learn-go/internal/domain"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain/coursework"
	"learn-go/internal/domain/curriculum"
	"learn-go/internal/domain/identity"
	"learn-go/internal/domain/roster"
	"learn-go/internal/domain/scheduling"
	"learn-go/internal/domain/storage"
)

var (
	// ErrStudentProfileNotFound 表示当前账号没有学生档案。
	ErrStudentProfileNotFound = errors.New("student profile not found")
	// ErrStudentReminderNotFound 表示找不到该学生的提醒事项。
	ErrStudentReminderNotFound = errors.New("student reminder not found")
	// ErrStudentReminderInvalid 表示创建或更新的数据无效。
	ErrStudentReminderInvalid = errors.New("student reminder invalid")
	// ErrCourseAccessDenied 表示学生无权访问该课程。
	ErrCourseAccessDenied = errors.New("course access denied")
)

// StudentPortalUsecase 为学生聚合教学相关数据。
type StudentPortalUsecase struct {
	students    roster.StudentRepository
	assignments coursework.AssignmentRepository
	submissions coursework.SubmissionRepository
	courses     curriculum.CourseRepository
	chapters    curriculum.CourseChapterRepository
	slots       scheduling.CourseSlotRepository
	sessions    scheduling.CourseSessionRepository
	teachers    roster.TeacherRepository
	accounts    identity.AccountRepository
	reminders   roster.StudentReminderRepository
}

// GetSchoolID 获取指定账号所属的学校 ID。
func (s *StudentPortalUsecase) GetSchoolID(ctx context.Context, accountID string) (string, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return "", err
	}
	if account == nil {
		return "", errors.New("account not found")
	}
	return account.SchoolID, nil
}

// GetStudentByAccountID 获取指定账号的学生档案。
func (s *StudentPortalUsecase) GetStudentByAccountID(ctx context.Context, accountID string) (*roster.Student, error) {
	return s.students.GetByAccountID(ctx, accountID)
}

// NewStudentPortalUsecase 构造 StudentPortalUsecase。
func NewStudentPortalUsecase(
	students roster.StudentRepository,
	assignments coursework.AssignmentRepository,
	submissions coursework.SubmissionRepository,
	courses curriculum.CourseRepository,
	chapters curriculum.CourseChapterRepository,
	slots scheduling.CourseSlotRepository,
	sessions scheduling.CourseSessionRepository,
	teachers roster.TeacherRepository,
	accounts identity.AccountRepository,
	reminders roster.StudentReminderRepository,
) *StudentPortalUsecase {
	return &StudentPortalUsecase{
		students:    students,
		assignments: assignments,
		submissions: submissions,
		courses:     courses,
		chapters:    chapters,
		slots:       slots,
		sessions:    sessions,
		teachers:    teachers,
		accounts:    accounts,
		reminders:   reminders,
	}
}

// StudentAssignmentItem 表示首页展示的作业/考试摘要。
type StudentAssignmentItem struct {
	ID            string                    `json:"id"`
	Title         string                    `json:"title"`
	Description   string                    `json:"description"`
	CourseID      string                    `json:"course_id"`
	CourseName    string                    `json:"course_name"`
	TeacherID     string                    `json:"teacher_id"`
	TeacherName   string                    `json:"teacher_name"`
	Type          coursework.AssignmentType `json:"type"`
	StartAt       *time.Time                `json:"start_at"`
	DueAt         *time.Time                `json:"due_at"`
	AllowResubmit bool                      `json:"allow_resubmit"`
	Status        string                    `json:"status"`
	SubmittedAt   *time.Time                `json:"submitted_at"`
	Score         *float64                  `json:"score"`
	Feedback      string                    `json:"feedback"`
	IsOverdue     bool                      `json:"is_overdue"`
}

// StudentScheduleItem 表示一次课次。
type StudentScheduleItem struct {
	SessionID   string    `json:"session_id"`
	CourseID    string    `json:"course_id"`
	CourseName  string    `json:"course_name"`
	TeacherID   string    `json:"teacher_id"`
	TeacherName string    `json:"teacher_name"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	Day         string    `json:"day"`
	SlotID      string    `json:"slot_id"`
	SlotName    string    `json:"slot_name"`
	Location    string    `json:"location"`
	Source      string    `json:"source"`
}

// AgendaItemKind 枚举日程事件类型。
type AgendaItemKind string

const (
	AgendaItemKindSession    AgendaItemKind = "session"
	AgendaItemKindAssignment AgendaItemKind = "assignment"
	AgendaItemKindExam       AgendaItemKind = "exam"
)

// StudentAgendaItem 表示合并后的日程视图。
type StudentAgendaItem struct {
	ID          string         `json:"id"`
	Kind        AgendaItemKind `json:"kind"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	CourseID    string         `json:"course_id"`
	CourseName  string         `json:"course_name"`
	TeacherID   string         `json:"teacher_id"`
	TeacherName string         `json:"teacher_name"`
	StartsAt    time.Time      `json:"starts_at"`
	EndsAt      *time.Time     `json:"ends_at"`
	Location    string         `json:"location"`
	Source      string         `json:"source"`
	Day         string         `json:"day"`
	Status      string         `json:"status"`
	SubmittedAt *time.Time     `json:"submitted_at"`
	Score       *float64       `json:"score"`
	Feedback    string         `json:"feedback"`
	IsOverdue   bool           `json:"is_overdue"`
}

// CreateStudentReminderInput 包含创建自定义提醒的字段。
type CreateStudentReminderInput struct {
	Title       string
	Description string
	TimeLabel   string
	Route       string
	Priority    roster.StudentReminderPriority
	Icon        string
}

// UpdateStudentReminderInput 包含更新提醒的可选字段。
type UpdateStudentReminderInput struct {
	Title       *string
	Description *string
	TimeLabel   *string
	Route       *string
	Priority    *roster.StudentReminderPriority
	Icon        *string
	Completed   *bool
}

// ListAssignments 返回学生所在班级的作业。
// 若提供了 courseID，则仅返回该课程的作业。
func (s *StudentPortalUsecase) ListAssignments(ctx context.Context, accountID string, limit int, courseID string) ([]StudentAssignmentItem, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}

	if student.ClassID == nil {
		return []StudentAssignmentItem{}, nil
	}

	assignments, err := s.assignments.ListByClass(ctx, *student.ClassID, limit, nil)
	if err != nil {
		return nil, err
	}

	if trimmed := strings.TrimSpace(courseID); trimmed != "" {
		filtered := make([]coursework.Assignment, 0, len(assignments))
		for _, a := range assignments {
			if a.CourseID == trimmed {
				filtered = append(filtered, a)
			}
		}
		assignments = filtered
	}

	return s.buildAssignmentItems(ctx, student.ID, assignments)
}

func (s *StudentPortalUsecase) ensureStudentCourseAccess(ctx context.Context, studentID string, courseID string) error {
	courses, err := s.courses.ListByStudentID(ctx, studentID)
	if err != nil {
		return err
	}
	for _, course := range courses {
		if course.ID == courseID {
			return nil
		}
	}
	return ErrCourseAccessDenied
}

// ListCourseChapters 列出学生所选课程的章节。
func (s *StudentPortalUsecase) ListCourseChapters(ctx context.Context, accountID string, courseID string) ([]curriculum.CourseChapter, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}
	if err := s.ensureStudentCourseAccess(ctx, student.ID, courseID); err != nil {
		return nil, err
	}
	return s.chapters.ListByCourse(ctx, courseID)
}

// GetCourseChapter 返回单个章节及其附件。
func (s *StudentPortalUsecase) GetCourseChapter(ctx context.Context, accountID string, courseID string, chapterID string) (*curriculum.CourseChapter, []storage.File, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}
	if student == nil {
		return nil, nil, ErrStudentProfileNotFound
	}
	if err := s.ensureStudentCourseAccess(ctx, student.ID, courseID); err != nil {
		return nil, nil, err
	}

	chapter, err := s.chapters.GetByID(ctx, courseID, chapterID)
	if err != nil {
		return nil, nil, err
	}
	files, err := s.chapters.ListFiles(ctx, chapterID)
	if err != nil {
		return nil, nil, err
	}
	return chapter, files, nil
}

// ListExams 返回学生所在班级的考试。
func (s *StudentPortalUsecase) ListExams(ctx context.Context, accountID string, limit int) ([]StudentAssignmentItem, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}

	if student.ClassID == nil {
		return []StudentAssignmentItem{}, nil
	}

	assignments, err := s.assignments.ListByClass(ctx, *student.ClassID, limit, []coursework.AssignmentType{coursework.AssignmentExam})
	if err != nil {
		return nil, err
	}
	return s.buildAssignmentItems(ctx, student.ID, assignments)
}

// ListSchedule 返回学生所在班级在指定时间窗口内的课次。
func (s *StudentPortalUsecase) ListSchedule(ctx context.Context, accountID string, start, end time.Time) ([]StudentScheduleItem, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}

	if student.ClassID == nil {
		return []StudentScheduleItem{}, nil
	}

	sessions, err := s.sessions.ListByClassBetween(ctx, *student.ClassID, start, end)
	if err != nil {
		return nil, err
	}

	return s.buildScheduleItems(ctx, sessions)
}

// ListAgenda 返回指定时间窗口内合并后的课程与作业日程。
func (s *StudentPortalUsecase) ListAgenda(ctx context.Context, accountID string, start, end time.Time, includeAssignments bool) ([]StudentAgendaItem, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}

	if student.ClassID == nil {
		return []StudentAgendaItem{}, nil
	}

	sessions, err := s.sessions.ListByClassBetween(ctx, *student.ClassID, start, end)
	if err != nil {
		return nil, err
	}
	scheduleItems, err := s.buildScheduleItems(ctx, sessions)
	if err != nil {
		return nil, err
	}

	agenda := make([]StudentAgendaItem, 0, len(scheduleItems))
	for _, session := range scheduleItems {
		agenda = append(agenda, StudentAgendaItem{
			ID:          session.SessionID,
			Kind:        AgendaItemKindSession,
			Title:       session.CourseName,
			Description: session.TeacherName,
			CourseID:    session.CourseID,
			CourseName:  session.CourseName,
			TeacherID:   session.TeacherID,
			TeacherName: session.TeacherName,
			StartsAt:    session.StartsAt,
			EndsAt:      &session.EndsAt,
			Location:    session.Location,
			Source:      session.Source,
			Day:         session.Day,
		})
	}

	if includeAssignments {
		assignments, err := s.assignments.ListDueBetween(ctx, *student.ClassID, start, end, nil)
		if err != nil {
			return nil, err
		}
		assignmentItems, err := s.buildAssignmentItems(ctx, student.ID, assignments)
		if err != nil {
			return nil, err
		}
		for _, assignment := range assignmentItems {
			var startAt time.Time
			if assignment.DueAt != nil {
				startAt = *assignment.DueAt
			} else if assignment.StartAt != nil {
				startAt = *assignment.StartAt
			} else {
				startAt = time.Now()
			}
			var endAt *time.Time
			if assignment.DueAt != nil {
				endAt = assignment.DueAt
			}
			kind := AgendaItemKindAssignment
			if assignment.Type == coursework.AssignmentExam {
				kind = AgendaItemKindExam
			}
			agenda = append(agenda, StudentAgendaItem{
				ID:          assignment.ID,
				Kind:        kind,
				Title:       assignment.Title,
				Description: assignment.Description,
				CourseID:    assignment.CourseID,
				CourseName:  assignment.CourseName,
				TeacherID:   assignment.TeacherID,
				TeacherName: assignment.TeacherName,
				StartsAt:    startAt,
				EndsAt:      endAt,
				Location:    "",
				Source:      "assignment",
				Day:         startAt.Format("2006-01-02"),
				Status:      assignment.Status,
				SubmittedAt: assignment.SubmittedAt,
				Score:       assignment.Score,
				Feedback:    assignment.Feedback,
				IsOverdue:   assignment.IsOverdue,
			})
		}
	}

	sort.SliceStable(agenda, func(i, j int) bool {
		if agenda[i].StartsAt.Equal(agenda[j].StartsAt) {
			return agenda[i].ID < agenda[j].ID
		}
		return agenda[i].StartsAt.Before(agenda[j].StartsAt)
	})

	return agenda, nil
}

func (s *StudentPortalUsecase) buildScheduleItems(ctx context.Context, sessions []scheduling.CourseSession) ([]StudentScheduleItem, error) {
	if len(sessions) == 0 {
		return []StudentScheduleItem{}, nil
	}
	courseNames, err := s.loadCourseNames(ctx, collectCourseIDsFromSessions(sessions))
	if err != nil {
		return nil, err
	}
	teacherNames, err := s.loadTeacherNames(ctx, collectTeacherIDsFromSessions(sessions))
	if err != nil {
		return nil, err
	}
	slotNames, err := s.loadSlotNames(ctx, collectSlotIDsFromSessions(sessions))
	if err != nil {
		return nil, err
	}

	items := make([]StudentScheduleItem, 0, len(sessions))
	for _, session := range sessions {
		teacherID := ""
		teacherName := ""
		if session.TeacherID != nil {
			teacherID = *session.TeacherID
			teacherName = teacherNames[teacherID]
		}

		items = append(items, StudentScheduleItem{
			SessionID:   session.ID,
			CourseID:    session.CourseID,
			CourseName:  courseNames[session.CourseID],
			TeacherID:   teacherID,
			TeacherName: teacherName,
			StartsAt:    session.StartsAt,
			EndsAt:      session.EndsAt,
			Day:         session.StartsAt.Format("2006-01-02"),
			SlotID:      session.SlotID,
			SlotName:    slotNames[session.SlotID],
			Location:    session.Location,
			Source:      session.Source,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].StartsAt.Equal(items[j].StartsAt) {
			return items[i].SessionID < items[j].SessionID
		}
		return items[i].StartsAt.Before(items[j].StartsAt)
	})

	return items, nil
}

func (s *StudentPortalUsecase) buildAssignmentItems(ctx context.Context, studentID string, assignments []coursework.Assignment) ([]StudentAssignmentItem, error) {
	if len(assignments) == 0 {
		return []StudentAssignmentItem{}, nil
	}

	assignmentIDs := make([]string, 0, len(assignments))
	courseIDs := make([]string, 0, len(assignments))
	teacherIDs := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		assignmentIDs = append(assignmentIDs, assignment.ID)
		courseIDs = append(courseIDs, assignment.CourseID)
		teacherIDs = append(teacherIDs, assignment.TeacherID)
	}

	submissions, err := s.submissions.ListByStudentAndAssignments(ctx, studentID, assignmentIDs)
	if err != nil {
		return nil, err
	}
	submissionMap := make(map[string]coursework.AssignmentSubmission, len(submissions))
	for _, submission := range submissions {
		submissionMap[submission.AssignmentID] = submission
	}

	courseNames, err := s.loadCourseNames(ctx, courseIDs)
	if err != nil {
		return nil, err
	}
	teacherNames, err := s.loadTeacherNames(ctx, teacherIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	items := make([]StudentAssignmentItem, 0, len(assignments))
	for _, assignment := range assignments {
		submission, hasSubmission := submissionMap[assignment.ID]
		status := "pending"
		if hasSubmission {
			switch submission.Status {
			case "graded":
				status = "graded"
			case "submitted":
				status = "submitted"
			default:
				status = submission.Status
			}
		}
		overdue := false
		if assignment.DueAt != nil && assignment.DueAt.Before(now) && status == "pending" {
			overdue = true
		}
		var submissionPtr *time.Time
		if hasSubmission && submission.SubmittedAt != nil {
			submissionPtr = submission.SubmittedAt
		}
		items = append(items, StudentAssignmentItem{
			ID:            assignment.ID,
			Title:         assignment.Title,
			Description:   assignment.Description,
			CourseID:      assignment.CourseID,
			CourseName:    courseNames[assignment.CourseID],
			TeacherID:     assignment.TeacherID,
			TeacherName:   teacherNames[assignment.TeacherID],
			Type:          assignment.Type,
			StartAt:       assignment.StartAt,
			DueAt:         assignment.DueAt,
			AllowResubmit: assignment.AllowResubmit,
			Status:        status,
			SubmittedAt:   submissionPtr,
			Score:         submission.Score,
			Feedback:      submission.Feedback,
			IsOverdue:     overdue,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		left := items[i].DueAt
		right := items[j].DueAt
		switch {
		case left == nil && right == nil:
			return items[i].Title < items[j].Title
		case left == nil:
			return false
		case right == nil:
			return true
		default:
			return left.Before(*right)
		}
	})

	return items, nil
}

func (s *StudentPortalUsecase) loadCourseNames(ctx context.Context, courseIDs []string) (map[string]string, error) {
	result := make(map[string]string)
	unique := uniqueStrings(courseIDs)
	if len(unique) == 0 {
		return result, nil
	}
	courses, err := s.courses.ListByIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	for _, course := range courses {
		result[course.ID] = course.Name
	}
	return result, nil
}

func (s *StudentPortalUsecase) loadTeacherNames(ctx context.Context, teacherIDs []string) (map[string]string, error) {
	result := make(map[string]string)
	unique := uniqueStrings(teacherIDs)
	if len(unique) == 0 {
		return result, nil
	}

	accountIDs := make([]string, 0, len(unique))
	teacherAccountMap := make(map[string]string, len(unique))
	for _, teacherID := range unique {
		teacher, err := s.teachers.GetByID(ctx, teacherID)
		if err != nil {
			return nil, err
		}
		teacherAccountMap[teacherID] = teacher.AccountID
		if teacher.AccountID != "" {
			accountIDs = append(accountIDs, teacher.AccountID)
		}
	}

	accounts, err := s.accounts.ListByIDs(ctx, accountIDs)
	if err != nil {
		return nil, err
	}
	nameByAccount := make(map[string]string, len(accounts))
	for _, account := range accounts {
		nameByAccount[account.ID] = account.DisplayName
	}

	for teacherID, accountID := range teacherAccountMap {
		if name, ok := nameByAccount[accountID]; ok {
			result[teacherID] = name
		}
	}
	return result, nil
}

func (s *StudentPortalUsecase) loadSlotNames(ctx context.Context, slotIDs []string) (map[string]string, error) {
	result := make(map[string]string)
	unique := uniqueStrings(slotIDs)
	if len(unique) == 0 {
		return result, nil
	}
	slots, err := s.slots.ListByIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	for _, slot := range slots {
		result[slot.ID] = slot.Name
	}
	return result, nil
}

func collectCourseIDsFromSessions(sessions []scheduling.CourseSession) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.CourseID)
	}
	return ids
}

func collectTeacherIDsFromSessions(sessions []scheduling.CourseSession) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		if session.TeacherID != nil {
			ids = append(ids, *session.TeacherID)
		}
	}
	return ids
}

func collectSlotIDsFromSessions(sessions []scheduling.CourseSession) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		if session.SlotID != "" {
			ids = append(ids, session.SlotID)
		}
	}
	return ids
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// ListCustomReminders 返回学生创建的自定义提醒。
func (s *StudentPortalUsecase) ListCustomReminders(ctx context.Context, accountID string) ([]roster.StudentReminder, error) {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.reminders.ListByStudent(ctx, student.ID)
}

// CreateCustomReminder 为学生保存一条新提醒。
func (s *StudentPortalUsecase) CreateCustomReminder(ctx context.Context, accountID string, input CreateStudentReminderInput) (*roster.StudentReminder, error) {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return nil, err
	}
	title, err := sanitizeReminderTitle(input.Title)
	if err != nil {
		return nil, err
	}
	reminder := &roster.StudentReminder{
		ID:          uuid.NewString(),
		StudentID:   student.ID,
		Title:       title,
		Description: sanitizeReminderDescription(input.Description),
		TimeLabel:   sanitizeReminderTimeLabel(input.TimeLabel),
		Route:       sanitizeReminderRoute(input.Route),
		Priority:    normalizeReminderPriority(input.Priority),
		Icon:        sanitizeReminderIcon(input.Icon),
	}
	if err := s.reminders.Create(ctx, reminder); err != nil {
		return nil, err
	}
	return reminder, nil
}

// UpdateCustomReminder 更新自定义提醒的可变字段。
func (s *StudentPortalUsecase) UpdateCustomReminder(ctx context.Context, accountID, reminderID string, input UpdateStudentReminderInput) (*roster.StudentReminder, error) {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return nil, err
	}
	updates := make(map[string]any)
	if input.Title != nil {
		title, err := sanitizeReminderTitle(*input.Title)
		if err != nil {
			return nil, err
		}
		updates["title"] = title
	}
	if input.Description != nil {
		updates["description"] = sanitizeReminderDescription(*input.Description)
	}
	if input.TimeLabel != nil {
		updates["time_label"] = sanitizeReminderTimeLabel(*input.TimeLabel)
	}
	if input.Route != nil {
		updates["route"] = sanitizeReminderRoute(*input.Route)
	}
	if input.Priority != nil {
		updates["priority"] = string(normalizeReminderPriority(*input.Priority))
	}
	if input.Icon != nil {
		updates["icon"] = sanitizeReminderIcon(*input.Icon)
	}
	if input.Completed != nil {
		if *input.Completed {
			updates["completed_at"] = time.Now()
		} else {
			updates["completed_at"] = nil
		}
	}
	if len(updates) == 0 {
		reminder, err := s.reminders.GetByID(ctx, reminderID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, ErrStudentReminderNotFound
			}
			return nil, err
		}
		if reminder.StudentID != student.ID {
			return nil, ErrStudentReminderNotFound
		}
		return reminder, nil
	}
	updated, err := s.reminders.UpdateFields(ctx, reminderID, student.ID, updates)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrStudentReminderNotFound
		}
		return nil, err
	}
	return updated, nil
}

// DeleteCustomReminder 删除学生拥有的提醒。
func (s *StudentPortalUsecase) DeleteCustomReminder(ctx context.Context, accountID, reminderID string) error {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return err
	}
	if err := s.reminders.Delete(ctx, reminderID, student.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrStudentReminderNotFound
		}
		return err
	}
	return nil
}

// MarkAllRemindersComplete 将学生的所有提醒标记为已完成。
func (s *StudentPortalUsecase) MarkAllRemindersComplete(ctx context.Context, accountID string) error {
	return s.SetAllRemindersCompletion(ctx, accountID, true)
}

// SetReminderCompletion 切换单条提醒的完成状态。
func (s *StudentPortalUsecase) SetReminderCompletion(ctx context.Context, accountID, reminderID string, completed bool) (*roster.StudentReminder, error) {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return nil, err
	}
	var ts *time.Time
	if completed {
		now := time.Now()
		ts = &now
	}
	updated, err := s.reminders.SetCompletion(ctx, reminderID, student.ID, completed, ts)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrStudentReminderNotFound
		}
		return nil, err
	}
	return updated, nil
}

// BatchSetReminderCompletion 批量切换多条提醒的完成状态。
func (s *StudentPortalUsecase) BatchSetReminderCompletion(ctx context.Context, accountID string, reminderIDs []string, completed bool) error {
	ids := uniqueStrings(reminderIDs)
	if len(ids) == 0 {
		return ErrStudentReminderInvalid
	}
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return err
	}
	var ts *time.Time
	if completed {
		now := time.Now()
		ts = &now
	}
	if err := s.reminders.MarkBatchCompleted(ctx, student.ID, ids, completed, ts); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrStudentReminderNotFound
		}
		return err
	}
	return nil
}

// SetAllRemindersCompletion 更新所有提醒的完成状态。
func (s *StudentPortalUsecase) SetAllRemindersCompletion(ctx context.Context, accountID string, completed bool) error {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return err
	}
	var ts *time.Time
	if completed {
		now := time.Now()
		ts = &now
	}
	return s.reminders.MarkAllCompleted(ctx, student.ID, completed, ts)
}

func (s *StudentPortalUsecase) currentStudent(ctx context.Context, accountID string) (*roster.Student, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}
	return student, nil
}

func sanitizeReminderTitle(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ErrStudentReminderInvalid
	}
	return trimmed, nil
}

func sanitizeReminderDescription(value string) string {
	return strings.TrimSpace(value)
}

func sanitizeReminderTimeLabel(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "时间待定"
	}
	return trimmed
}

func sanitizeReminderRoute(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return trimmed
}

func normalizeReminderPriority(value roster.StudentReminderPriority) roster.StudentReminderPriority {
	switch value {
	case roster.StudentReminderPriorityHigh:
		return roster.StudentReminderPriorityHigh
	default:
		return roster.StudentReminderPriorityNormal
	}
}

var allowedReminderIcons = map[string]struct{}{
	"assignment": {},
	"voice":      {},
	"book":       {},
	"note":       {},
	"alarm":      {},
	"ai":         {},
}

func sanitizeReminderIcon(value string) string {
	icon := strings.ToLower(strings.TrimSpace(value))
	if _, ok := allowedReminderIcons[icon]; ok {
		return icon
	}
	return "alarm"
}

// GetStudentProfile 对外暴露 currentStudent。
func (s *StudentPortalUsecase) GetStudentProfile(ctx context.Context, accountID string) (*roster.Student, error) {
	return s.currentStudent(ctx, accountID)
}
