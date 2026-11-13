package usecase

import (
	"context"
	"errors"
	"fmt"
	"learn-go/internal/domain"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain/coursework"
	"learn-go/internal/domain/identity"
	"learn-go/internal/domain/notification"
	"learn-go/internal/domain/roster"
	"learn-go/internal/domain/storage"
)

// AssignmentUsecase 管理作业与提交。
type AssignmentUsecase struct {
	assignments   coursework.AssignmentRepository
	submissions   coursework.SubmissionRepository
	comments      coursework.SubmissionCommentRepository
	students      roster.StudentRepository
	notifications *NotificationUsecase
	files         *FileUsecase
}

// NewAssignmentUsecase 创建 AssignmentUsecase。
func NewAssignmentUsecase(
	assignments coursework.AssignmentRepository,
	submissions coursework.SubmissionRepository,
	comments coursework.SubmissionCommentRepository,
	students roster.StudentRepository,
	notifications *NotificationUsecase,
	files *FileUsecase,
) *AssignmentUsecase {
	return &AssignmentUsecase{
		assignments:   assignments,
		submissions:   submissions,
		comments:      comments,
		students:      students,
		notifications: notifications,
		files:         files,
	}
}

// ErrAssignmentNotFound 表示作业不存在。
var ErrAssignmentNotFound = errors.New("assignment not found")

// ErrSubmissionNotFound 表示提交记录不存在。
var ErrSubmissionNotFound = errors.New("submission not found")

// ErrSubmissionForbidden 表示调用方无权访问该提交。
var ErrSubmissionForbidden = errors.New("submission forbidden")

// ErrSubmissionAlreadyGraded 表示已批改的提交不允许学生修改。
var ErrSubmissionAlreadyGraded = errors.New("submission already graded")

// ErrScoreOutOfRange 表示分数超出允许范围。
var ErrScoreOutOfRange = errors.New("score out of allowed range")

// CreateAssignmentInput 包含创建作业所需的数据。
type CreateAssignmentInput struct {
	CourseID      string
	TeacherID     string
	ClassID       string
	Type          coursework.AssignmentType
	Title         string
	Description   string
	StartAt       *time.Time
	DueAt         *time.Time
	MaxScore      float64
	AllowResubmit bool
	Questions     []QuestionInput
	Attachments   []string
}

// QuestionInput 描述一道题目。
type QuestionInput struct {
	Type       coursework.QuestionType
	Prompt     string
	Options    string
	Answer     string
	Score      float64
	OrderIndex int
}

// CreateAssignment 创建作业及其题目。
func (s *AssignmentUsecase) CreateAssignment(ctx context.Context, input CreateAssignmentInput) (*coursework.Assignment, error) {
	if input.CourseID == "" || input.TeacherID == "" || input.ClassID == "" {
		return nil, errors.New("course, teacher and class are required")
	}
	if err := validateAssignmentQuestionScores(input.MaxScore, input.Questions); err != nil {
		return nil, err
	}

	assignment := &coursework.Assignment{
		ID:            uuid.NewString(),
		CourseID:      input.CourseID,
		TeacherID:     input.TeacherID,
		ClassID:       input.ClassID,
		Type:          input.Type,
		Title:         input.Title,
		Description:   input.Description,
		StartAt:       input.StartAt,
		DueAt:         input.DueAt,
		MaxScore:      input.MaxScore,
		AllowResubmit: input.AllowResubmit,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	questions := make([]coursework.AssignmentQuestion, 0, len(input.Questions))
	for _, q := range input.Questions {
		questions = append(questions, coursework.AssignmentQuestion{
			ID:         uuid.NewString(),
			Type:       q.Type,
			Prompt:     q.Prompt,
			Options:    q.Options,
			Answer:     q.Answer,
			Score:      q.Score,
			OrderIndex: q.OrderIndex,
		})
	}

	attachments := make([]coursework.AssignmentAttachment, 0, len(input.Attachments))
	for _, fileID := range input.Attachments {
		attachments = append(attachments, coursework.AssignmentAttachment{
			ID:           uuid.NewString(),
			AssignmentID: assignment.ID,
			FileID:       fileID,
			CreatedAt:    time.Now(),
		})
	}

	if err := s.assignments.Create(ctx, assignment, questions, attachments); err != nil {
		return nil, err
	}

	// 通知学生
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		students, err := s.students.ListByClassID(bgCtx, input.ClassID)
		if err == nil {
			for _, student := range students {
				s.notifications.Create(bgCtx, student.AccountID, "New Assignment: "+input.Title, "A new assignment has been published.", notification.NotificationTypeAssignment, assignment.ID)
			}
		}
	}()

	return assignment, nil
}

// SubmitAssignmentInput 表示学生提交的作答数据。
type SubmitAssignmentInput struct {
	AssignmentID string
	StudentID    string
	Answers      []AnswerInput
	Score        *float64
	Feedback     string
	Status       string
}

// AnswerInput 将答案与题目对应。
type AnswerInput struct {
	QuestionID string
	Answer     string
	Score      *float64
}

// Submit 记录或更新学生的提交。
func (s *AssignmentUsecase) Submit(ctx context.Context, input SubmitAssignmentInput) error {
	if input.AssignmentID == "" || input.StudentID == "" {
		return errors.New("assignment and student required")
	}

	// 计算完成进度
	assignment, questions, _, err := s.assignments.Get(ctx, input.AssignmentID)
	if err != nil {
		return err
	}
	if err := validateBoundedScore(input.Score, assignment.MaxScore); err != nil {
		return err
	}

	questionIDs := make(map[string]bool)
	questionMaxScores := make(map[string]float64, len(questions))
	for _, q := range questions {
		questionIDs[q.ID] = true
		questionMaxScores[q.ID] = q.Score
	}

	answeredCount := 0
	seenQuestions := make(map[string]bool)
	for _, ans := range input.Answers {
		if err := validateBoundedScore(ans.Score, questionMaxScores[ans.QuestionID]); err != nil {
			if !questionIDs[ans.QuestionID] {
				return errors.New("invalid question id")
			}
			return err
		}
		if questionIDs[ans.QuestionID] && strings.TrimSpace(ans.Answer) != "" {
			if !seenQuestions[ans.QuestionID] {
				answeredCount++
				seenQuestions[ans.QuestionID] = true
			}
		}
	}

	progress := 0
	if len(questions) > 0 {
		progress = (answeredCount * 100) / len(questions)
		if progress > 100 {
			progress = 100
		}
	}

	submission := &coursework.AssignmentSubmission{
		ID:           uuid.NewString(),
		AssignmentID: input.AssignmentID,
		StudentID:    input.StudentID,
		Score:        input.Score,
		Feedback:     input.Feedback,
		Status:       input.Status,
		Progress:     progress,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if input.Status == "submitted" {
		now := time.Now()
		submission.SubmittedAt = &now
	}

	// 检查提交是否已存在，以保留原 ID
	existing, _, err := s.submissions.GetByAssignmentAndStudent(ctx, input.AssignmentID, input.StudentID)
	if err == nil && existing != nil {
		if existing.Status == "graded" {
			return ErrSubmissionAlreadyGraded
		}
		submission.ID = existing.ID
		submission.CreatedAt = existing.CreatedAt
	}

	items := make([]coursework.SubmissionItem, 0, len(input.Answers))
	for _, ans := range input.Answers {
		items = append(items, coursework.SubmissionItem{
			ID:         uuid.NewString(),
			QuestionID: ans.QuestionID,
			Answer:     ans.Answer,
			Score:      ans.Score,
		})
	}

	return s.submissions.CreateOrUpdate(ctx, submission, items)
}

// SubmissionDetail 聚合一次提交及其各题作答。
type SubmissionDetail struct {
	Submission  coursework.AssignmentSubmission
	Items       []coursework.SubmissionItem
	StudentName string
}

// GetAssignment 查询作业及其题目。
func (s *AssignmentUsecase) GetAssignment(ctx context.Context, assignmentID string) (*coursework.Assignment, []coursework.AssignmentQuestion, []storage.File, error) {
	if assignmentID == "" {
		return nil, nil, nil, errors.New("assignment id required")
	}
	assignment, questions, files, err := s.assignments.Get(ctx, assignmentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, nil, ErrAssignmentNotFound
		}
		return nil, nil, nil, err
	}

	return assignment, questions, files, nil
}

// ListAssignmentSubmissions 返回某作业的所有提交及各题作答。
func (s *AssignmentUsecase) ListAssignmentSubmissions(ctx context.Context, assignmentID string) ([]SubmissionDetail, error) {
	if assignmentID == "" {
		return nil, errors.New("assignment id required")
	}

	subs, err := s.submissions.ListByAssignment(ctx, assignmentID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(subs))
	studentIDs := make([]string, 0, len(subs))
	for _, sub := range subs {
		ids = append(ids, sub.ID)
		studentIDs = append(studentIDs, sub.StudentID)
	}

	items, err := s.submissions.ListItemsBySubmissionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	students, err := s.students.ListByIDs(ctx, studentIDs)
	if err != nil {
		return nil, err
	}
	studentNameMap := make(map[string]string, len(students))
	for _, st := range students {
		studentNameMap[st.ID] = st.Name
	}

	itemsBySubmission := make(map[string][]coursework.SubmissionItem, len(subs))
	for _, item := range items {
		itemsBySubmission[item.SubmissionID] = append(itemsBySubmission[item.SubmissionID], item)
	}

	details := make([]SubmissionDetail, 0, len(subs))
	for _, sub := range subs {
		details = append(details, SubmissionDetail{
			Submission:  sub,
			Items:       itemsBySubmission[sub.ID],
			StudentName: studentNameMap[sub.StudentID],
		})
	}
	return details, nil
}

// GetSubmissionForStudent 返回学生的提交，包含各题作答与评论。
func (s *AssignmentUsecase) GetSubmissionForStudent(ctx context.Context, assignmentID, studentID string) (*SubmissionDetail, []coursework.SubmissionComment, error) {
	if assignmentID == "" || studentID == "" {
		return nil, nil, errors.New("assignment and student required")
	}

	if _, _, _, err := s.GetAssignment(ctx, assignmentID); err != nil {
		return nil, nil, err
	}

	submission, items, err := s.submissions.GetByAssignmentAndStudent(ctx, assignmentID, studentID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrSubmissionNotFound
		}
		return nil, nil, err
	}

	comments, err := s.comments.ListBySubmission(ctx, submission.ID)
	if err != nil {
		return nil, nil, err
	}

	return &SubmissionDetail{Submission: *submission, Items: items}, comments, nil
}

// GetSubmissionForTeacher 返回提交记录，并确保该作业属于当前教师。
func (s *AssignmentUsecase) GetSubmissionForTeacher(ctx context.Context, teacherID, assignmentID, submissionID string) (*SubmissionDetail, []coursework.SubmissionComment, error) {
	if teacherID == "" || assignmentID == "" || submissionID == "" {
		return nil, nil, errors.New("teacher, assignment and submission required")
	}

	assignment, _, _, err := s.GetAssignment(ctx, assignmentID)
	if err != nil {
		return nil, nil, err
	}
	if assignment.TeacherID != teacherID {
		return nil, nil, ErrSubmissionForbidden
	}

	submission, items, err := s.submissions.GetByID(ctx, submissionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrSubmissionNotFound
		}
		return nil, nil, err
	}
	if submission.AssignmentID != assignmentID {
		return nil, nil, ErrSubmissionForbidden
	}

	comments, err := s.comments.ListBySubmission(ctx, submission.ID)
	if err != nil {
		return nil, nil, err
	}

	return &SubmissionDetail{Submission: *submission, Items: items}, comments, nil
}

// SubmissionCommentInput 表示提交评论的数据。
type SubmissionCommentInput struct {
	Content string
}

// GradeSubmissionInput 表示批改更新的数据。
type GradeSubmissionInput struct {
	AssignmentID string
	SubmissionID string
	AccountID    string
	Score        *float64
	Feedback     string
	ItemScores   map[string]*float64
	Comment      *SubmissionCommentInput
}

// GradeSubmission 更新评分并可附带评论。
func (s *AssignmentUsecase) GradeSubmission(ctx context.Context, teacherID string, input GradeSubmissionInput) (*SubmissionDetail, []coursework.SubmissionComment, error) {
	if teacherID == "" {
		return nil, nil, errors.New("teacher id required")
	}
	if input.AssignmentID == "" || input.SubmissionID == "" {
		return nil, nil, errors.New("assignment and submission required")
	}

	assignment, questions, _, err := s.GetAssignment(ctx, input.AssignmentID)
	if err != nil {
		return nil, nil, err
	}
	if assignment.TeacherID != teacherID {
		return nil, nil, ErrSubmissionForbidden
	}
	if err := validateBoundedScore(input.Score, assignment.MaxScore); err != nil {
		return nil, nil, err
	}

	questionMaxScores := make(map[string]float64, len(questions))
	for _, q := range questions {
		questionMaxScores[q.ID] = q.Score
	}

	submission, items, err := s.submissions.GetByID(ctx, input.SubmissionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrSubmissionNotFound
		}
		return nil, nil, err
	}
	if submission.AssignmentID != input.AssignmentID {
		return nil, nil, ErrSubmissionForbidden
	}

	itemByID := make(map[string]coursework.SubmissionItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}

	updates := make([]coursework.SubmissionItem, 0, len(input.ItemScores))
	for itemID, score := range input.ItemScores {
		original, ok := itemByID[itemID]
		if !ok {
			return nil, nil, errors.New("invalid item id")
		}
		maxScore, ok := questionMaxScores[original.QuestionID]
		if !ok {
			return nil, nil, errors.New("invalid question id")
		}
		if err := validateBoundedScore(score, maxScore); err != nil {
			return nil, nil, err
		}
		original.Score = score
		updates = append(updates, original)
	}

	if input.Score != nil {
		submission.Score = input.Score
	}
	if input.Feedback != "" {
		submission.Feedback = input.Feedback
	}
	if submission.Status != "graded" {
		submission.Status = "graded"
	}
	submission.UpdatedAt = time.Now()

	if err := s.submissions.UpdateGrades(ctx, submission, updates); err != nil {
		return nil, nil, err
	}

	if input.Comment != nil && strings.TrimSpace(input.Comment.Content) != "" {
		if input.AccountID == "" {
			return nil, nil, errors.New("account id required for comment")
		}
		comment := &coursework.SubmissionComment{
			ID:           uuid.NewString(),
			SubmissionID: submission.ID,
			AuthorID:     input.AccountID,
			AuthorRole:   identity.RoleTeacher,
			Content:      input.Comment.Content,
			CreatedAt:    time.Now(),
		}
		if err := s.comments.Create(ctx, comment); err != nil {
			return nil, nil, err
		}
	}

	comments, err := s.comments.ListBySubmission(ctx, submission.ID)
	if err != nil {
		return nil, nil, err
	}

	mergedItems := mergeItems(items, updates)
	return &SubmissionDetail{Submission: *submission, Items: mergedItems}, comments, nil
}

// ReturnSubmissionInput 表示退回提交的数据。
type ReturnSubmissionInput struct {
	AssignmentID string
	SubmissionID string
	AccountID    string
	Comment      string
}

// ReturnSubmission 将提交退回给学生重做。
func (s *AssignmentUsecase) ReturnSubmission(ctx context.Context, teacherID string, input ReturnSubmissionInput) (*SubmissionDetail, []coursework.SubmissionComment, error) {
	if teacherID == "" {
		return nil, nil, errors.New("teacher id required")
	}
	if input.AssignmentID == "" || input.SubmissionID == "" {
		return nil, nil, errors.New("assignment and submission required")
	}

	assignment, _, _, err := s.GetAssignment(ctx, input.AssignmentID)
	if err != nil {
		return nil, nil, err
	}
	if assignment.TeacherID != teacherID {
		return nil, nil, ErrSubmissionForbidden
	}

	submission, items, err := s.submissions.GetByID(ctx, input.SubmissionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, ErrSubmissionNotFound
		}
		return nil, nil, err
	}
	if submission.AssignmentID != input.AssignmentID {
		return nil, nil, ErrSubmissionForbidden
	}

	submission.Status = "returned"
	if strings.TrimSpace(input.Comment) != "" {
		submission.Feedback = input.Comment
	}
	submission.UpdatedAt = time.Now()

	// 这里使用 UpdateGrades，因为即使没有作答项它也会更新提交记录
	if err := s.submissions.UpdateGrades(ctx, submission, []coursework.SubmissionItem{}); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Comment) != "" {
		if input.AccountID == "" {
			return nil, nil, errors.New("account id required for comment")
		}
		comment := &coursework.SubmissionComment{
			ID:           uuid.NewString(),
			SubmissionID: submission.ID,
			AuthorID:     input.AccountID,
			AuthorRole:   identity.RoleTeacher,
			Content:      input.Comment,
			CreatedAt:    time.Now(),
		}
		if err := s.comments.Create(ctx, comment); err != nil {
			return nil, nil, err
		}
	}

	comments, err := s.comments.ListBySubmission(ctx, submission.ID)
	if err != nil {
		return nil, nil, err
	}

	return &SubmissionDetail{Submission: *submission, Items: items}, comments, nil
}

// UpdateAssignmentInput 包含更新作业所需的数据。
type UpdateAssignmentInput struct {
	ID            string
	TeacherID     string
	Title         *string
	Description   *string
	StartAt       *time.Time
	DueAt         *time.Time
	MaxScore      *float64
	AllowResubmit *bool
}

// UpdateAssignment 更新已有作业。
func (s *AssignmentUsecase) UpdateAssignment(ctx context.Context, input UpdateAssignmentInput) (*coursework.Assignment, error) {
	if input.ID == "" || input.TeacherID == "" {
		return nil, errors.New("assignment id and teacher id required")
	}

	assignment, questions, _, err := s.assignments.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if assignment.TeacherID != input.TeacherID {
		return nil, errors.New("forbidden")
	}

	if input.Title != nil {
		assignment.Title = *input.Title
	}
	if input.Description != nil {
		assignment.Description = *input.Description
	}
	if input.StartAt != nil {
		assignment.StartAt = input.StartAt
	}
	if input.DueAt != nil {
		assignment.DueAt = input.DueAt
	}
	if input.MaxScore != nil {
		if err := validateFiniteNonNegative(*input.MaxScore); err != nil {
			return nil, err
		}
		totalQuestionScore := 0.0
		for _, q := range questions {
			totalQuestionScore += q.Score
		}
		if totalQuestionScore > *input.MaxScore {
			return nil, fmt.Errorf("%w: total question score %.2f exceeds assignment max_score %.2f", ErrScoreOutOfRange, totalQuestionScore, *input.MaxScore)
		}
		assignment.MaxScore = *input.MaxScore
	}
	if input.AllowResubmit != nil {
		assignment.AllowResubmit = *input.AllowResubmit
	}
	assignment.UpdatedAt = time.Now()

	if err := s.assignments.Update(ctx, assignment); err != nil {
		return nil, err
	}

	return assignment, nil
}

func validateAssignmentQuestionScores(maxScore float64, questions []QuestionInput) error {
	if err := validateFiniteNonNegative(maxScore); err != nil {
		return err
	}
	total := 0.0
	for _, q := range questions {
		if err := validateFiniteNonNegative(q.Score); err != nil {
			return err
		}
		if q.Score > maxScore {
			return fmt.Errorf("%w: question score %.2f exceeds assignment max_score %.2f", ErrScoreOutOfRange, q.Score, maxScore)
		}
		total += q.Score
	}
	if total > maxScore {
		return fmt.Errorf("%w: total question score %.2f exceeds assignment max_score %.2f", ErrScoreOutOfRange, total, maxScore)
	}
	return nil
}

func validateBoundedScore(score *float64, max float64) error {
	if score == nil {
		return nil
	}
	if err := validateFiniteNonNegative(*score); err != nil {
		return err
	}
	if err := validateFiniteNonNegative(max); err != nil {
		return err
	}
	if *score > max {
		return fmt.Errorf("%w: score %.2f exceeds max %.2f", ErrScoreOutOfRange, *score, max)
	}
	return nil
}

func validateFiniteNonNegative(v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("%w: score must be a finite number", ErrScoreOutOfRange)
	}
	if v < 0 {
		return fmt.Errorf("%w: score must be >= 0", ErrScoreOutOfRange)
	}
	return nil
}

func mergeItems(original []coursework.SubmissionItem, updates []coursework.SubmissionItem) []coursework.SubmissionItem {
	if len(updates) == 0 {
		return original
	}
	updated := make(map[string]coursework.SubmissionItem, len(updates))
	for _, item := range updates {
		updated[item.ID] = item
	}
	result := make([]coursework.SubmissionItem, 0, len(original))
	for _, item := range original {
		if upd, ok := updated[item.ID]; ok {
			result = append(result, upd)
		} else {
			result = append(result, item)
		}
	}
	return result
}
