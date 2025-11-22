package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
)

var (
	// ErrStudentProfileNotFound indicates the current account has no student profile.
	ErrStudentProfileNotFound = errors.New("student profile not found")
	// ErrStudentReminderNotFound indicates the reminder cannot be located for the student.
	ErrStudentReminderNotFound = errors.New("student reminder not found")
	// ErrStudentReminderInvalid indicates create/update payload is invalid.
	ErrStudentReminderInvalid = errors.New("student reminder invalid")
)

// StudentPortalService aggregates academic data for students.
type StudentPortalService struct {
	students    repository.StudentRepository
	assignments repository.AssignmentRepository
	submissions repository.SubmissionRepository
	courses     repository.CourseRepository
	slots       repository.CourseSlotRepository
	sessions    repository.CourseSessionRepository
	teachers    repository.TeacherRepository
	accounts    repository.AccountRepository
	reminders   repository.StudentReminderRepository
}

// NewStudentPortalService constructs StudentPortalService.
func NewStudentPortalService(
	students repository.StudentRepository,
	assignments repository.AssignmentRepository,
	submissions repository.SubmissionRepository,
	courses repository.CourseRepository,
	slots repository.CourseSlotRepository,
	sessions repository.CourseSessionRepository,
	teachers repository.TeacherRepository,
	accounts repository.AccountRepository,
	reminders repository.StudentReminderRepository,
) *StudentPortalService {
	return &StudentPortalService{
		students:    students,
		assignments: assignments,
		submissions: submissions,
		courses:     courses,
		slots:       slots,
		sessions:    sessions,
		teachers:    teachers,
		accounts:    accounts,
		reminders:   reminders,
	}
}

// StudentAssignmentItem represents homework/exam summary for dashboard.
type StudentAssignmentItem struct {
	ID            string                `json:"id"`
	Title         string                `json:"title"`
	Description   string                `json:"description"`
	CourseID      string                `json:"course_id"`
	CourseName    string                `json:"course_name"`
	TeacherID     string                `json:"teacher_id"`
	TeacherName   string                `json:"teacher_name"`
	Type          domain.AssignmentType `json:"type"`
	StartAt       *time.Time            `json:"start_at"`
	DueAt         *time.Time            `json:"due_at"`
	AllowResubmit bool                  `json:"allow_resubmit"`
	Status        string                `json:"status"`
	SubmittedAt   *time.Time            `json:"submitted_at"`
	Score         *float64              `json:"score"`
	Feedback      string                `json:"feedback"`
	IsOverdue     bool                  `json:"is_overdue"`
}

// StudentScheduleItem represents a single course session.
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

// AgendaItemKind enumerates agenda event types.
type AgendaItemKind string

const (
	AgendaItemKindSession    AgendaItemKind = "session"
	AgendaItemKindAssignment AgendaItemKind = "assignment"
	AgendaItemKindExam       AgendaItemKind = "exam"
)

// StudentAgendaItem represents a merged schedule view.
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

// CreateStudentReminderInput contains fields for creating custom reminders.
type CreateStudentReminderInput struct {
	Title       string
	Description string
	TimeLabel   string
	Route       string
	Priority    domain.StudentReminderPriority
	Icon        string
}

// UpdateStudentReminderInput contains optional fields for reminder updates.
type UpdateStudentReminderInput struct {
	Title       *string
	Description *string
	TimeLabel   *string
	Route       *string
	Priority    *domain.StudentReminderPriority
	Icon        *string
	Completed   *bool
}

// ListAssignments returns assignments for the student's class.
func (s *StudentPortalService) ListAssignments(ctx context.Context, accountID string, limit int) ([]StudentAssignmentItem, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}

	assignments, err := s.assignments.ListByClass(ctx, student.ClassID, limit, nil)
	if err != nil {
		return nil, err
	}
	return s.buildAssignmentItems(ctx, student.ID, assignments)
}

// ListExams returns exam-type assignments for the student's class.
func (s *StudentPortalService) ListExams(ctx context.Context, accountID string, limit int) ([]StudentAssignmentItem, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}

	assignments, err := s.assignments.ListByClass(ctx, student.ClassID, limit, []domain.AssignmentType{domain.AssignmentExam})
	if err != nil {
		return nil, err
	}
	return s.buildAssignmentItems(ctx, student.ID, assignments)
}

// ListSchedule returns course sessions for the student's class within the given window.
func (s *StudentPortalService) ListSchedule(ctx context.Context, accountID string, start, end time.Time) ([]StudentScheduleItem, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}

	sessions, err := s.sessions.ListByClassBetween(ctx, student.ClassID, start, end)
	if err != nil {
		return nil, err
	}

	return s.buildScheduleItems(ctx, sessions)
}

// ListAgenda returns merged lesson & assignment schedule within window.
func (s *StudentPortalService) ListAgenda(ctx context.Context, accountID string, start, end time.Time, includeAssignments bool) ([]StudentAgendaItem, error) {
	student, err := s.students.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if student == nil {
		return nil, ErrStudentProfileNotFound
	}

	sessions, err := s.sessions.ListByClassBetween(ctx, student.ClassID, start, end)
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
		assignments, err := s.assignments.ListDueBetween(ctx, student.ClassID, start, end, nil)
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
			if assignment.Type == domain.AssignmentExam {
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

func (s *StudentPortalService) buildScheduleItems(ctx context.Context, sessions []domain.CourseSession) ([]StudentScheduleItem, error) {
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
		items = append(items, StudentScheduleItem{
			SessionID:   session.ID,
			CourseID:    session.CourseID,
			CourseName:  courseNames[session.CourseID],
			TeacherID:   session.TeacherID,
			TeacherName: teacherNames[session.TeacherID],
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

func (s *StudentPortalService) buildAssignmentItems(ctx context.Context, studentID string, assignments []domain.Assignment) ([]StudentAssignmentItem, error) {
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
	submissionMap := make(map[string]domain.AssignmentSubmission, len(submissions))
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

func (s *StudentPortalService) loadCourseNames(ctx context.Context, courseIDs []string) (map[string]string, error) {
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

func (s *StudentPortalService) loadTeacherNames(ctx context.Context, teacherIDs []string) (map[string]string, error) {
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

func (s *StudentPortalService) loadSlotNames(ctx context.Context, slotIDs []string) (map[string]string, error) {
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

func collectCourseIDsFromSessions(sessions []domain.CourseSession) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.CourseID)
	}
	return ids
}

func collectTeacherIDsFromSessions(sessions []domain.CourseSession) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.TeacherID)
	}
	return ids
}

func collectSlotIDsFromSessions(sessions []domain.CourseSession) []string {
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

// ListCustomReminders returns custom reminders created by the student.
func (s *StudentPortalService) ListCustomReminders(ctx context.Context, accountID string) ([]domain.StudentReminder, error) {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.reminders.ListByStudent(ctx, student.ID)
}

// CreateCustomReminder stores a new reminder for the student.
func (s *StudentPortalService) CreateCustomReminder(ctx context.Context, accountID string, input CreateStudentReminderInput) (*domain.StudentReminder, error) {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return nil, err
	}
	title, err := sanitizeReminderTitle(input.Title)
	if err != nil {
		return nil, err
	}
	reminder := &domain.StudentReminder{
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

// UpdateCustomReminder updates mutable fields of a custom reminder.
func (s *StudentPortalService) UpdateCustomReminder(ctx context.Context, accountID, reminderID string, input UpdateStudentReminderInput) (*domain.StudentReminder, error) {
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
			if errors.Is(err, gorm.ErrRecordNotFound) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrStudentReminderNotFound
		}
		return nil, err
	}
	return updated, nil
}

// DeleteCustomReminder removes a reminder owned by the student.
func (s *StudentPortalService) DeleteCustomReminder(ctx context.Context, accountID, reminderID string) error {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return err
	}
	if err := s.reminders.Delete(ctx, reminderID, student.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrStudentReminderNotFound
		}
		return err
	}
	return nil
}

// MarkAllRemindersComplete marks all reminders as completed for the student.
func (s *StudentPortalService) MarkAllRemindersComplete(ctx context.Context, accountID string) error {
	student, err := s.currentStudent(ctx, accountID)
	if err != nil {
		return err
	}
	now := time.Now()
	return s.reminders.MarkAllCompleted(ctx, student.ID, true, &now)
}

func (s *StudentPortalService) currentStudent(ctx context.Context, accountID string) (*domain.Student, error) {
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

func normalizeReminderPriority(value domain.StudentReminderPriority) domain.StudentReminderPriority {
	switch value {
	case domain.StudentReminderPriorityHigh:
		return domain.StudentReminderPriorityHigh
	default:
		return domain.StudentReminderPriorityNormal
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
