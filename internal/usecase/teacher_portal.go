package service

import (
	"context"
	"errors"
	assignmentbiz "learn-go/internal/biz/assignment"
	coursebiz "learn-go/internal/biz/course"
	identitybiz "learn-go/internal/biz/identity"
	schoolbiz "learn-go/internal/biz/school"
	storagebiz "learn-go/internal/biz/storage"
	"sort"
	"time"
)

// ErrTeacherProfileNotFound indicates missing teacher profile for account.
var ErrTeacherProfileNotFound = errors.New("teacher profile not found")

// ErrTeacherCourseAccessDenied indicates the teacher doesn't have access to the course.
var ErrTeacherCourseAccessDenied = errors.New("teacher course access denied")

// ErrTeacherAssignmentForbidden indicates the assignment does not belong to the teacher.
var ErrTeacherAssignmentForbidden = errors.New("teacher assignment forbidden")

// TeacherPortalService aggregates schedule and assignment data for teachers.
type TeacherPortalService struct {
	teachers       identitybiz.TeacherRepository
	assignments    assignmentbiz.AssignmentRepository
	submissions    assignmentbiz.SubmissionRepository
	students       identitybiz.StudentRepository
	sessions       coursebiz.CourseSessionRepository
	courses        coursebiz.CourseRepository
	classes        schoolbiz.ClassRepository
	slots          coursebiz.CourseSlotRepository
	accounts       identitybiz.AccountRepository
	schedules      coursebiz.CourseScheduleRepository
	chapters       coursebiz.CourseChapterRepository
	courseStudents coursebiz.CourseStudentRepository
	courseTeachers coursebiz.CourseTeacherRepository
}

// GetSchoolID retrieves the school ID for a given account.
func (s *TeacherPortalService) GetSchoolID(ctx context.Context, accountID string) (string, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return "", err
	}
	if account == nil {
		return "", errors.New("account not found")
	}
	return account.SchoolID, nil
}

// GetProfile retrieves the teacher profile for a given account.
func (s *TeacherPortalService) GetProfile(ctx context.Context, accountID string) (*identitybiz.Teacher, error) {
	return s.teachers.GetByAccountID(ctx, accountID)
}

// NewTeacherPortalService constructs TeacherPortalService.
func NewTeacherPortalService(
	teachers identitybiz.TeacherRepository,
	assignments assignmentbiz.AssignmentRepository,
	submissions assignmentbiz.SubmissionRepository,
	students identitybiz.StudentRepository,
	sessions coursebiz.CourseSessionRepository,
	courses coursebiz.CourseRepository,
	classes schoolbiz.ClassRepository,
	slots coursebiz.CourseSlotRepository,
	accounts identitybiz.AccountRepository,
	schedules coursebiz.CourseScheduleRepository,
	chapters coursebiz.CourseChapterRepository,
	courseStudents coursebiz.CourseStudentRepository,
	courseTeachers coursebiz.CourseTeacherRepository,
) *TeacherPortalService {
	return &TeacherPortalService{
		teachers:       teachers,
		assignments:    assignments,
		submissions:    submissions,
		students:       students,
		sessions:       sessions,
		courses:        courses,
		classes:        classes,
		slots:          slots,
		accounts:       accounts,
		schedules:      schedules,
		chapters:       chapters,
		courseStudents: courseStudents,
		courseTeachers: courseTeachers,
	}
}

// ensureTeacherCourseAccess validates that the teacher owns or teaches the course.
func (s *TeacherPortalService) ensureTeacherCourseAccess(ctx context.Context, accountID, courseID string) (*identitybiz.Teacher, *coursebiz.Course, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, nil, err
	}
	if teacher == nil {
		return nil, nil, ErrTeacherProfileNotFound
	}

	course, err := s.courses.GetByID(ctx, courseID)
	if err != nil {
		return nil, nil, err
	}

	if course != nil && course.SchoolID != "" && teacher.SchoolID != "" && course.SchoolID != teacher.SchoolID {
		return nil, nil, ErrTeacherCourseAccessDenied
	}

	allowed, err := s.hasCourseAccess(ctx, teacher.ID, courseID)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, ErrTeacherCourseAccessDenied
	}

	return teacher, course, nil
}

func (s *TeacherPortalService) hasCourseAccess(ctx context.Context, teacherID, courseID string) (bool, error) {
	if s.courseTeachers != nil {
		ids, err := s.courseTeachers.ListCourseIDsByTeacher(ctx, teacherID)
		if err != nil {
			return false, err
		}
		for _, id := range ids {
			if id == courseID {
				return true, nil
			}
		}
	}

	if s.schedules != nil {
		schedules, err := s.schedules.ListByTeacher(ctx, teacherID)
		if err != nil {
			return false, err
		}
		for _, sch := range schedules {
			if sch.CourseID == courseID {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *TeacherPortalService) GetAssignedCourses(ctx context.Context, accountID string) ([]coursebiz.Course, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	courseIDs := make(map[string]bool)

	if s.schedules != nil {
		schedules, err := s.schedules.ListByTeacher(ctx, teacher.ID)
		if err != nil {
			return nil, err
		}
		for _, sch := range schedules {
			courseIDs[sch.CourseID] = true
		}
	}

	if s.courseTeachers != nil {
		ids, err := s.courseTeachers.ListCourseIDsByTeacher(ctx, teacher.ID)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			courseIDs[id] = true
		}
	}

	if len(courseIDs) == 0 {
		return []coursebiz.Course{}, nil
	}

	flat := make([]string, 0, len(courseIDs))
	for id := range courseIDs {
		flat = append(flat, id)
	}

	courses, err := s.courses.ListByIDs(ctx, flat)
	if err != nil {
		return nil, err
	}
	for i := range courses {
		courses[i].ImageURL = normalizeCourseImageURL(courses[i].ImageURL)
	}
	return courses, nil
}

func (s *TeacherPortalService) GetCourseClasses(ctx context.Context, accountID, courseID string) ([]schoolbiz.Class, error) {
	teacher, course, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return nil, err
	}

	classIDs := make(map[string]bool)

	if s.schedules != nil {
		schedules, err := s.schedules.ListByTeacher(ctx, teacher.ID)
		if err != nil {
			return nil, err
		}
		for _, sch := range schedules {
			if sch.CourseID == courseID {
				classIDs[sch.ClassID] = true
			}
		}
	}

	if s.courseStudents != nil {
		enrollments, err := s.courseStudents.ListByCourseID(ctx, courseID)
		if err != nil {
			return nil, err
		}
		studentIDs := make([]string, 0, len(enrollments))
		for _, e := range enrollments {
			studentIDs = append(studentIDs, e.StudentID)
		}
		if len(studentIDs) > 0 {
			students, err := s.students.ListByIDs(ctx, studentIDs)
			if err != nil {
				return nil, err
			}
			for _, st := range students {
				if st.ClassID != nil {
					classIDs[*st.ClassID] = true
				}
			}
		}
	}

	flat := make([]string, 0, len(classIDs))
	for id := range classIDs {
		flat = append(flat, id)
	}

	if len(flat) == 0 {
		return []schoolbiz.Class{}, nil
	}

	classes, err := s.classes.ListByIDs(ctx, flat)
	if err != nil {
		return nil, err
	}

	filtered := make([]schoolbiz.Class, 0, len(classes))
	for _, cls := range classes {
		if cls.SchoolID != "" && course != nil && course.SchoolID != "" && cls.SchoolID != course.SchoolID {
			continue
		}
		filtered = append(filtered, cls)
	}

	return filtered, nil
}

func (s *TeacherPortalService) GetClassStudents(ctx context.Context, classID string) ([]identitybiz.Student, error) {
	return s.students.ListByClassID(ctx, classID)
}

// TeacherScheduleItem represents a planned lesson for a teacher.
type TeacherScheduleItem struct {
	SessionID  string    `json:"session_id"`
	CourseID   string    `json:"course_id"`
	CourseName string    `json:"course_name"`
	ClassID    string    `json:"class_id"`
	ClassName  string    `json:"class_name"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	Day        string    `json:"day"`
	SlotID     string    `json:"slot_id"`
	SlotName   string    `json:"slot_name"`
	Location   string    `json:"location"`
	Source     string    `json:"source"`
}

// TeacherAssignmentItem summarizes an assignment created by a teacher.
type TeacherAssignmentItem struct {
	ID                 string                       `json:"id"`
	Title              string                       `json:"title"`
	Description        string                       `json:"description"`
	CourseID           string                       `json:"course_id"`
	CourseName         string                       `json:"course_name"`
	ClassID            string                       `json:"class_id"`
	ClassName          string                       `json:"class_name"`
	Type               assignmentbiz.AssignmentType `json:"type"`
	StartAt            *time.Time                   `json:"start_at"`
	DueAt              *time.Time                   `json:"due_at"`
	AllowResubmit      bool                         `json:"allow_resubmit"`
	SubmissionCount    int                          `json:"submission_count"`
	SubmittedCount     int                          `json:"submitted_count"`
	ClassStudentCount  int                          `json:"class_student_count"`
	GradedCount        int                          `json:"graded_count"`
	PendingGradeCount  int                          `json:"pending_grade_count"`
	LatestSubmissionAt *time.Time                   `json:"latest_submission_at"`
	MissingCount       int                          `json:"missing_count"`
	ScoreAverage       *float64                     `json:"score_average"`
	ScoreMax           *float64                     `json:"score_max"`
	ScoreMin           *float64                     `json:"score_min"`
	ScoreDistribution  TeacherScoreDistribution     `json:"score_distribution"`
}

// TeacherAssignmentDetail provides full assignment context for teachers.
type TeacherAssignmentDetail struct {
	ID            string                       `json:"id"`
	Title         string                       `json:"title"`
	Description   string                       `json:"description"`
	CourseID      string                       `json:"course_id"`
	CourseName    string                       `json:"course_name"`
	ClassID       string                       `json:"class_id"`
	ClassName     string                       `json:"class_name"`
	Type          assignmentbiz.AssignmentType `json:"type"`
	StartAt       *time.Time                   `json:"start_at"`
	DueAt         *time.Time                   `json:"due_at"`
	AllowResubmit bool                         `json:"allow_resubmit"`
	MaxScore      float64                      `json:"max_score"`
	Questions     []TeacherQuestionItem        `json:"questions"`
	Stats         TeacherAssignmentStats       `json:"stats"`
	Attachments   []storagebiz.File            `json:"attachments"`
}

// TeacherQuestionItem mirrors assignment questions.
type TeacherQuestionItem struct {
	ID         string                     `json:"id"`
	Type       assignmentbiz.QuestionType `json:"type"`
	Prompt     string                     `json:"prompt"`
	Options    string                     `json:"options,omitempty"`
	Answer     string                     `json:"answer,omitempty"`
	Score      float64                    `json:"score"`
	OrderIndex int                        `json:"order_index"`
}

// TeacherAssignmentStats aggregates submission metrics.
type TeacherAssignmentStats struct {
	SubmissionCount    int                      `json:"submission_count"`
	SubmittedCount     int                      `json:"submitted_count"`
	GradedCount        int                      `json:"graded_count"`
	PendingGradeCount  int                      `json:"pending_grade_count"`
	MissingCount       int                      `json:"missing_count"`
	ClassStudentCount  int                      `json:"class_student_count"`
	ScoreAverage       *float64                 `json:"score_average"`
	ScoreMax           *float64                 `json:"score_max"`
	ScoreMin           *float64                 `json:"score_min"`
	LatestSubmissionAt *time.Time               `json:"latest_submission_at"`
	ScoreDistribution  TeacherScoreDistribution `json:"score_distribution"`
}

// TeacherScoreDistribution exposes graded score buckets.
type TeacherScoreDistribution struct {
	Below60        int `json:"below_60"`
	Between60And70 int `json:"between_60_70"`
	Between70And80 int `json:"between_70_80"`
	Between80And90 int `json:"between_80_90"`
	Above90        int `json:"above_90"`
}

// TeacherAssignmentGradeExport describes grade export metadata and rows.
type TeacherAssignmentGradeExport struct {
	AssignmentID    string                            `json:"assignment_id"`
	AssignmentTitle string                            `json:"assignment_title"`
	CourseID        string                            `json:"course_id"`
	CourseName      string                            `json:"course_name"`
	ClassID         string                            `json:"class_id"`
	ClassName       string                            `json:"class_name"`
	Type            assignmentbiz.AssignmentType      `json:"type"`
	MaxScore        float64                           `json:"max_score"`
	Rows            []TeacherAssignmentGradeExportRow `json:"rows"`
}

// TeacherAssignmentGradeExportRow represents a student's grade record.
type TeacherAssignmentGradeExportRow struct {
	StudentID     string     `json:"student_id"`
	StudentNumber string     `json:"student_number"`
	StudentName   string     `json:"student_name"`
	Status        string     `json:"status"`
	Score         *float64   `json:"score"`
	SubmittedAt   *time.Time `json:"submitted_at"`
}

// TeacherAgendaItem merges sessions and assignment deadlines.
type TeacherAgendaItem struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	CourseID    string     `json:"course_id"`
	CourseName  string     `json:"course_name"`
	ClassID     string     `json:"class_id"`
	ClassName   string     `json:"class_name"`
	StartsAt    time.Time  `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	Day         string     `json:"day"`
	SlotID      string     `json:"slot_id"`
	SlotName    string     `json:"slot_name"`
	Location    string     `json:"location"`
	Source      string     `json:"source"`
}

// ListSchedule returns teaching sessions for the authenticated teacher.
func (s *TeacherPortalService) ListSchedule(ctx context.Context, accountID string, start, end time.Time) ([]TeacherScheduleItem, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	sessions, err := s.sessions.ListByTeacherBetween(ctx, teacher.ID, start, end)
	if err != nil {
		return nil, err
	}
	return s.buildScheduleItems(ctx, sessions)
}

// ListAssignments returns assignments authored by the teacher.
func (s *TeacherPortalService) ListAssignments(ctx context.Context, accountID string, limit int, classID string, types []assignmentbiz.AssignmentType) ([]TeacherAssignmentItem, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	assignments, err := s.assignments.ListByTeacher(ctx, teacher.ID, limit, classID, types)
	if err != nil {
		return nil, err
	}
	return s.buildAssignmentItems(ctx, assignments)
}

// ListExams returns exam-type assignments authored by the teacher.
func (s *TeacherPortalService) ListExams(ctx context.Context, accountID string, limit int, classID string) ([]TeacherAssignmentItem, error) {
	return s.ListAssignments(ctx, accountID, limit, classID, []assignmentbiz.AssignmentType{assignmentbiz.AssignmentExam})
}

// ListAgenda combines schedule sessions and assignment deadlines.
func (s *TeacherPortalService) ListAgenda(ctx context.Context, accountID string, start, end time.Time, includeAssignments bool) ([]TeacherAgendaItem, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	sessions, err := s.sessions.ListByTeacherBetween(ctx, teacher.ID, start, end)
	if err != nil {
		return nil, err
	}
	scheduleItems, err := s.buildScheduleItems(ctx, sessions)
	if err != nil {
		return nil, err
	}

	agenda := make([]TeacherAgendaItem, 0, len(scheduleItems))
	for _, session := range scheduleItems {
		end := session.EndsAt
		agenda = append(agenda, TeacherAgendaItem{
			ID:          session.SessionID,
			Kind:        "session",
			Title:       session.CourseName,
			Description: session.ClassName,
			CourseID:    session.CourseID,
			CourseName:  session.CourseName,
			ClassID:     session.ClassID,
			ClassName:   session.ClassName,
			StartsAt:    session.StartsAt,
			EndsAt:      &end,
			Day:         session.Day,
			SlotID:      session.SlotID,
			SlotName:    session.SlotName,
			Location:    session.Location,
			Source:      session.Source,
		})
	}

	if includeAssignments {
		assignments, err := s.assignments.ListDueBetweenByTeacher(ctx, teacher.ID, start, end, nil)
		if err != nil {
			return nil, err
		}
		assignmentItems, err := s.buildAssignmentItems(ctx, assignments)
		if err != nil {
			return nil, err
		}
		for _, item := range assignmentItems {
			var startAt time.Time
			if item.DueAt != nil {
				startAt = *item.DueAt
			} else if item.StartAt != nil {
				startAt = *item.StartAt
			} else {
				startAt = time.Now()
			}
			var endAt *time.Time
			if item.DueAt != nil {
				endAt = item.DueAt
			}
			agenda = append(agenda, TeacherAgendaItem{
				ID:          item.ID,
				Kind:        string(item.Type),
				Title:       item.Title,
				Description: item.Description,
				CourseID:    item.CourseID,
				CourseName:  item.CourseName,
				ClassID:     item.ClassID,
				ClassName:   item.ClassName,
				StartsAt:    startAt,
				EndsAt:      endAt,
				Day:         startAt.Format("2006-01-02"),
				Location:    "",
				Source:      "assignment",
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

// GetAssignmentDetail returns a fully-hydrated assignment for the teacher.
func (s *TeacherPortalService) GetAssignmentDetail(ctx context.Context, accountID, assignmentID string, includeAnswers bool) (*TeacherAssignmentDetail, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	assignment, questions, files, err := s.assignments.Get(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment.TeacherID != teacher.ID {
		return nil, ErrTeacherAssignmentForbidden
	}

	courseNames, err := s.loadCourseNames(ctx, []string{assignment.CourseID})
	if err != nil {
		return nil, err
	}
	classNames, err := s.loadClassNames(ctx, []string{assignment.ClassID})
	if err != nil {
		return nil, err
	}
	statsMap, err := s.loadSubmissionStats(ctx, []string{assignment.ID})
	if err != nil {
		return nil, err
	}
	classCounts, err := s.loadClassStudentCounts(ctx, []string{assignment.ClassID})
	if err != nil {
		return nil, err
	}

	stat := statsMap[assignment.ID]
	classSize := classCounts[assignment.ClassID]
	pending := int(stat.Submitted - stat.Graded)
	if pending < 0 {
		pending = 0
	}
	missing := int(classSize - stat.Submitted)
	if missing < 0 {
		missing = 0
	}

	questionsDTO := make([]TeacherQuestionItem, 0, len(questions))
	for _, q := range questions {
		item := TeacherQuestionItem{
			ID:         q.ID,
			Type:       q.Type,
			Prompt:     q.Prompt,
			Score:      q.Score,
			OrderIndex: q.OrderIndex,
		}
		if includeAnswers {
			item.Options = q.Options
			item.Answer = q.Answer
		}
		questionsDTO = append(questionsDTO, item)
	}

	return &TeacherAssignmentDetail{
		ID:            assignment.ID,
		Title:         assignment.Title,
		Description:   assignment.Description,
		CourseID:      assignment.CourseID,
		CourseName:    courseNames[assignment.CourseID],
		ClassID:       assignment.ClassID,
		ClassName:     classNames[assignment.ClassID],
		Type:          assignment.Type,
		StartAt:       assignment.StartAt,
		DueAt:         assignment.DueAt,
		AllowResubmit: assignment.AllowResubmit,
		MaxScore:      assignment.MaxScore,
		Questions:     questionsDTO,
		Stats: TeacherAssignmentStats{
			SubmissionCount:    int(stat.Total),
			SubmittedCount:     int(stat.Submitted),
			GradedCount:        int(stat.Graded),
			PendingGradeCount:  pending,
			MissingCount:       missing,
			ClassStudentCount:  int(classSize),
			ScoreAverage:       stat.AverageScore,
			ScoreMax:           stat.MaxScore,
			ScoreMin:           stat.MinScore,
			LatestSubmissionAt: stat.LatestSubmittedAt,
			ScoreDistribution:  toTeacherScoreDistribution(stat.ScoreDistribution),
		},
		Attachments: files,
	}, nil
}

// ExportAssignmentGrades aggregates submission rows for CSV export.
func (s *TeacherPortalService) ExportAssignmentGrades(ctx context.Context, accountID, assignmentID string) (*TeacherAssignmentGradeExport, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	assignment, _, _, err := s.assignments.Get(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if assignment.TeacherID != teacher.ID {
		return nil, ErrTeacherAssignmentForbidden
	}

	submissions, err := s.submissions.ListByAssignment(ctx, assignment.ID)
	if err != nil {
		return nil, err
	}

	studentIDs := make([]string, 0, len(submissions))
	for _, submission := range submissions {
		studentIDs = append(studentIDs, submission.StudentID)
	}
	studentsByID, err := s.loadStudentProfiles(ctx, studentIDs)
	if err != nil {
		return nil, err
	}

	accountIDs := make([]string, 0, len(studentsByID))
	for _, student := range studentsByID {
		if student != nil && student.AccountID != "" {
			accountIDs = append(accountIDs, student.AccountID)
		}
	}
	accountsByID, err := s.loadAccounts(ctx, accountIDs)
	if err != nil {
		return nil, err
	}

	rows := make([]TeacherAssignmentGradeExportRow, 0, len(submissions))
	for _, submission := range submissions {
		row := TeacherAssignmentGradeExportRow{
			StudentID:   submission.StudentID,
			Status:      submission.Status,
			Score:       submission.Score,
			SubmittedAt: submission.SubmittedAt,
		}
		if student, ok := studentsByID[submission.StudentID]; ok && student != nil {
			row.StudentNumber = student.Number
			if account, ok := accountsByID[student.AccountID]; ok && account != nil {
				row.StudentName = account.DisplayName
			}
		}
		rows = append(rows, row)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		left := rows[i].StudentNumber
		right := rows[j].StudentNumber
		switch {
		case left == "" && right == "":
			return rows[i].StudentID < rows[j].StudentID
		case left == "":
			return false
		case right == "":
			return true
		default:
			if left == right {
				return rows[i].StudentID < rows[j].StudentID
			}
			return left < right
		}
	})

	courseNames, err := s.loadCourseNames(ctx, []string{assignment.CourseID})
	if err != nil {
		return nil, err
	}
	classNames, err := s.loadClassNames(ctx, []string{assignment.ClassID})
	if err != nil {
		return nil, err
	}

	return &TeacherAssignmentGradeExport{
		AssignmentID:    assignment.ID,
		AssignmentTitle: assignment.Title,
		CourseID:        assignment.CourseID,
		CourseName:      courseNames[assignment.CourseID],
		ClassID:         assignment.ClassID,
		ClassName:       classNames[assignment.ClassID],
		Type:            assignment.Type,
		MaxScore:        assignment.MaxScore,
		Rows:            rows,
	}, nil
}

func (s *TeacherPortalService) buildScheduleItems(ctx context.Context, sessions []coursebiz.CourseSession) ([]TeacherScheduleItem, error) {
	if len(sessions) == 0 {
		return []TeacherScheduleItem{}, nil
	}
	courseNames, err := s.loadCourseNames(ctx, collectCourseIDsFromSessions(sessions))
	if err != nil {
		return nil, err
	}
	classNames, err := s.loadClassNames(ctx, collectClassIDsFromSessions(sessions))
	if err != nil {
		return nil, err
	}
	slotNames, err := s.loadSlotNames(ctx, collectSlotIDsFromSessions(sessions))
	if err != nil {
		return nil, err
	}

	items := make([]TeacherScheduleItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, TeacherScheduleItem{
			SessionID:  session.ID,
			CourseID:   session.CourseID,
			CourseName: courseNames[session.CourseID],
			ClassID:    session.ClassID,
			ClassName:  classNames[session.ClassID],
			StartsAt:   session.StartsAt,
			EndsAt:     session.EndsAt,
			Day:        session.StartsAt.Format("2006-01-02"),
			SlotID:     session.SlotID,
			SlotName:   slotNames[session.SlotID],
			Location:   session.Location,
			Source:     session.Source,
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

func (s *TeacherPortalService) buildAssignmentItems(ctx context.Context, assignments []assignmentbiz.Assignment) ([]TeacherAssignmentItem, error) {
	if len(assignments) == 0 {
		return []TeacherAssignmentItem{}, nil
	}
	stats, err := s.loadSubmissionStats(ctx, collectAssignmentIDs(assignments))
	if err != nil {
		return nil, err
	}
	classStudentCounts, err := s.loadClassStudentCounts(ctx, collectClassIDsFromAssignments(assignments))
	if err != nil {
		return nil, err
	}
	courseNames, err := s.loadCourseNames(ctx, collectCourseIDsFromAssignments(assignments))
	if err != nil {
		return nil, err
	}
	classNames, err := s.loadClassNames(ctx, collectClassIDsFromAssignments(assignments))
	if err != nil {
		return nil, err
	}

	items := make([]TeacherAssignmentItem, 0, len(assignments))
	for _, assignment := range assignments {
		stat := stats[assignment.ID]
		pending := int(stat.Submitted - stat.Graded)
		if pending < 0 {
			pending = 0
		}
		classSize := classStudentCounts[assignment.ClassID]
		missing := int(classSize - stat.Submitted)
		if missing < 0 {
			missing = 0
		}
		items = append(items, TeacherAssignmentItem{
			ID:                 assignment.ID,
			Title:              assignment.Title,
			Description:        assignment.Description,
			CourseID:           assignment.CourseID,
			CourseName:         courseNames[assignment.CourseID],
			ClassID:            assignment.ClassID,
			ClassName:          classNames[assignment.ClassID],
			Type:               assignment.Type,
			StartAt:            assignment.StartAt,
			DueAt:              assignment.DueAt,
			AllowResubmit:      assignment.AllowResubmit,
			SubmissionCount:    int(stat.Total),
			SubmittedCount:     int(stat.Submitted),
			ClassStudentCount:  int(classSize),
			GradedCount:        int(stat.Graded),
			PendingGradeCount:  pending,
			LatestSubmissionAt: stat.LatestSubmittedAt,
			MissingCount:       missing,
			ScoreAverage:       stat.AverageScore,
			ScoreMax:           stat.MaxScore,
			ScoreMin:           stat.MinScore,
			ScoreDistribution:  toTeacherScoreDistribution(stat.ScoreDistribution),
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

// SearchAssignments finds assignments by keyword.
func (s *TeacherPortalService) SearchAssignments(ctx context.Context, accountID string, query string) ([]TeacherAssignmentItem, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	assignments, err := s.assignments.Search(ctx, teacher.ID, query)
	if err != nil {
		return nil, err
	}
	return s.buildAssignmentItems(ctx, assignments)
}

func (s *TeacherPortalService) loadCourseNames(ctx context.Context, ids []string) (map[string]string, error) {
	result := make(map[string]string)
	unique := uniqueStrings(ids)
	if len(unique) == 0 {
		return result, nil
	}
	courses, err := s.courses.ListByIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	for _, c := range courses {
		result[c.ID] = c.Name
	}
	return result, nil
}

func (s *TeacherPortalService) loadClassNames(ctx context.Context, ids []string) (map[string]string, error) {
	result := make(map[string]string)
	unique := uniqueStrings(ids)
	if len(unique) == 0 {
		return result, nil
	}
	classes, err := s.classes.ListByIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	for _, class := range classes {
		result[class.ID] = class.Name
	}
	return result, nil
}

func (s *TeacherPortalService) loadSlotNames(ctx context.Context, ids []string) (map[string]string, error) {
	result := make(map[string]string)
	unique := uniqueStrings(ids)
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

func (s *TeacherPortalService) loadStudentProfiles(ctx context.Context, ids []string) (map[string]*identitybiz.Student, error) {
	result := make(map[string]*identitybiz.Student)
	unique := uniqueStrings(ids)
	if len(unique) == 0 {
		return result, nil
	}
	students, err := s.students.ListByIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	for i := range students {
		student := students[i]
		copy := student
		result[student.ID] = &copy
	}
	return result, nil
}

// GetTeacherID resolves the teacher ID for a given account ID.
func (s *TeacherPortalService) GetTeacherID(ctx context.Context, accountID string) (string, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return "", err
	}
	if teacher == nil {
		return "", ErrTeacherProfileNotFound
	}
	return teacher.ID, nil
}

func (s *TeacherPortalService) loadAccounts(ctx context.Context, ids []string) (map[string]*identitybiz.Account, error) {
	result := make(map[string]*identitybiz.Account)
	if s.accounts == nil {
		return result, nil
	}
	unique := uniqueStrings(ids)
	if len(unique) == 0 {
		return result, nil
	}
	accounts, err := s.accounts.ListByIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		account := accounts[i]
		copy := account
		result[account.ID] = &copy
	}
	return result, nil
}

func collectCourseIDsFromAssignments(assignments []assignmentbiz.Assignment) []string {
	ids := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.CourseID)
	}
	return ids
}

func collectClassIDsFromAssignments(assignments []assignmentbiz.Assignment) []string {
	ids := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.ClassID)
	}
	return ids
}

func collectClassIDsFromSessions(sessions []coursebiz.CourseSession) []string {
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.ClassID)
	}
	return ids
}

func collectAssignmentIDs(assignments []assignmentbiz.Assignment) []string {
	ids := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.ID)
	}
	return ids
}

func toTeacherScoreDistribution(dist assignmentbiz.AssignmentScoreDistribution) TeacherScoreDistribution {
	return TeacherScoreDistribution{
		Below60:        int(dist.Below60),
		Between60And70: int(dist.Between60And70),
		Between70And80: int(dist.Between70And80),
		Between80And90: int(dist.Between80And90),
		Above90:        int(dist.Above90),
	}
}

func (s *TeacherPortalService) loadSubmissionStats(ctx context.Context, assignmentIDs []string) (map[string]assignmentbiz.AssignmentSubmissionStats, error) {
	result := make(map[string]assignmentbiz.AssignmentSubmissionStats)
	unique := uniqueStrings(assignmentIDs)
	if len(unique) == 0 {
		return result, nil
	}
	stats, err := s.submissions.StatsByAssignments(ctx, unique)
	if err != nil {
		return nil, err
	}
	for _, stat := range stats {
		result[stat.AssignmentID] = stat
	}
	return result, nil
}

func (s *TeacherPortalService) loadClassStudentCounts(ctx context.Context, classIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	unique := uniqueStrings(classIDs)
	if len(unique) == 0 {
		return result, nil
	}
	counts, err := s.students.CountByClassIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	for classID, count := range counts {
		result[classID] = count
	}
	return result, nil
}
