package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// AssignmentService manages assignments and submissions.
type AssignmentService struct {
	assignments   repository.AssignmentRepository
	submissions   repository.SubmissionRepository
	comments      repository.SubmissionCommentRepository
	students      repository.StudentRepository
	notifications *NotificationService
	files         *FileService
}

// NewAssignmentService creates a new AssignmentService.
func NewAssignmentService(
	assignments repository.AssignmentRepository,
	submissions repository.SubmissionRepository,
	comments repository.SubmissionCommentRepository,
	students repository.StudentRepository,
	notifications *NotificationService,
	files *FileService,
) *AssignmentService {
	return &AssignmentService{
		assignments:   assignments,
		submissions:   submissions,
		comments:      comments,
		students:      students,
		notifications: notifications,
		files:         files,
	}
}

// ErrAssignmentNotFound indicates the assignment does not exist.
var ErrAssignmentNotFound = errors.New("assignment not found")

// ErrSubmissionNotFound indicates the submission does not exist.
var ErrSubmissionNotFound = errors.New("submission not found")

// ErrSubmissionForbidden indicates the caller cannot access the submission.
var ErrSubmissionForbidden = errors.New("submission forbidden")

// ErrScoreOutOfRange indicates a score exceeds its allowed range.
var ErrScoreOutOfRange = errors.New("score out of allowed range")

// CreateAssignmentInput contains data for creating an assignment.
type CreateAssignmentInput struct {
	CourseID      string
	TeacherID     string
	ClassID       string
	Type          domain.AssignmentType
	Title         string
	Description   string
	StartAt       *time.Time
	DueAt         *time.Time
	MaxScore      float64
	AllowResubmit bool
	Questions     []QuestionInput
	Attachments   []string
}

// QuestionInput describes a single question.
type QuestionInput struct {
	Type       domain.QuestionType
	Prompt     string
	Options    string
	Answer     string
	Score      float64
	OrderIndex int
}

// CreateAssignment creates an assignment with its questions.
func (s *AssignmentService) CreateAssignment(ctx context.Context, input CreateAssignmentInput) (*domain.Assignment, error) {
	if input.CourseID == "" || input.TeacherID == "" || input.ClassID == "" {
		return nil, errors.New("course, teacher and class are required")
	}
	if err := validateAssignmentQuestionScores(input.MaxScore, input.Questions); err != nil {
		return nil, err
	}

	assignment := &domain.Assignment{
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

	questions := make([]domain.AssignmentQuestion, 0, len(input.Questions))
	for _, q := range input.Questions {
		questions = append(questions, domain.AssignmentQuestion{
			ID:         uuid.NewString(),
			Type:       q.Type,
			Prompt:     q.Prompt,
			Options:    q.Options,
			Answer:     q.Answer,
			Score:      q.Score,
			OrderIndex: q.OrderIndex,
		})
	}

	attachments := make([]domain.AssignmentAttachment, 0, len(input.Attachments))
	for _, fileID := range input.Attachments {
		attachments = append(attachments, domain.AssignmentAttachment{
			ID:           uuid.NewString(),
			AssignmentID: assignment.ID,
			FileID:       fileID,
			CreatedAt:    time.Now(),
		})
	}

	if err := s.assignments.Create(ctx, assignment, questions, attachments); err != nil {
		return nil, err
	}

	// Notify students
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		students, err := s.students.ListByClassID(bgCtx, input.ClassID)
		if err == nil {
			for _, student := range students {
				s.notifications.Create(bgCtx, student.AccountID, "New Assignment: "+input.Title, "A new assignment has been published.", domain.NotificationTypeAssignment, assignment.ID)
			}
		}
	}()

	return assignment, nil
}

// SubmitAssignmentInput captures student submission payload.
type SubmitAssignmentInput struct {
	AssignmentID string
	StudentID    string
	Answers      []AnswerInput
	Score        *float64
	Feedback     string
	Status       string
}

// AnswerInput ties an answer to a question.
type AnswerInput struct {
	QuestionID string
	Answer     string
	Score      *float64
}

// Submit records or updates a student submission.
func (s *AssignmentService) Submit(ctx context.Context, input SubmitAssignmentInput) error {
	if input.AssignmentID == "" || input.StudentID == "" {
		return errors.New("assignment and student required")
	}

	// Calculate progress
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

	submission := &domain.AssignmentSubmission{
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

	// Check if submission already exists to preserve ID
	existing, _, err := s.submissions.GetByAssignmentAndStudent(ctx, input.AssignmentID, input.StudentID)
	if err == nil && existing != nil {
		submission.ID = existing.ID
		submission.CreatedAt = existing.CreatedAt
	}

	items := make([]domain.SubmissionItem, 0, len(input.Answers))
	for _, ans := range input.Answers {
		items = append(items, domain.SubmissionItem{
			ID:         uuid.NewString(),
			QuestionID: ans.QuestionID,
			Answer:     ans.Answer,
			Score:      ans.Score,
		})
	}

	return s.submissions.CreateOrUpdate(ctx, submission, items)
}

// SubmissionDetail aggregates a submission and its items.
type SubmissionDetail struct {
	Submission  domain.AssignmentSubmission
	Items       []domain.SubmissionItem
	StudentName string
}

// GetAssignment retrieves an assignment with its questions.
func (s *AssignmentService) GetAssignment(ctx context.Context, assignmentID string) (*domain.Assignment, []domain.AssignmentQuestion, []domain.File, error) {
	if assignmentID == "" {
		return nil, nil, nil, errors.New("assignment id required")
	}
	assignment, questions, files, err := s.assignments.Get(ctx, assignmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrAssignmentNotFound
		}
		return nil, nil, nil, err
	}

	return assignment, questions, files, nil
}

// ListAssignmentSubmissions returns submissions and items for an assignment.
func (s *AssignmentService) ListAssignmentSubmissions(ctx context.Context, assignmentID string) ([]SubmissionDetail, error) {
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

	itemsBySubmission := make(map[string][]domain.SubmissionItem, len(subs))
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

// GetSubmissionForStudent returns the student's submission with items and comments.
func (s *AssignmentService) GetSubmissionForStudent(ctx context.Context, assignmentID, studentID string) (*SubmissionDetail, []domain.SubmissionComment, error) {
	if assignmentID == "" || studentID == "" {
		return nil, nil, errors.New("assignment and student required")
	}

	if _, _, _, err := s.GetAssignment(ctx, assignmentID); err != nil {
		return nil, nil, err
	}

	submission, items, err := s.submissions.GetByAssignmentAndStudent(ctx, assignmentID, studentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

// GetSubmissionForTeacher returns a submission ensuring the teacher owns the assignment.
func (s *AssignmentService) GetSubmissionForTeacher(ctx context.Context, teacherID, assignmentID, submissionID string) (*SubmissionDetail, []domain.SubmissionComment, error) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

// SubmissionCommentInput captures submission comment payload.
type SubmissionCommentInput struct {
	Content string
}

// GradeSubmissionInput captures grading updates.
type GradeSubmissionInput struct {
	AssignmentID string
	SubmissionID string
	AccountID    string
	Score        *float64
	Feedback     string
	ItemScores   map[string]*float64
	Comment      *SubmissionCommentInput
}

// GradeSubmission applies scoring updates and optional comment.
func (s *AssignmentService) GradeSubmission(ctx context.Context, teacherID string, input GradeSubmissionInput) (*SubmissionDetail, []domain.SubmissionComment, error) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrSubmissionNotFound
		}
		return nil, nil, err
	}
	if submission.AssignmentID != input.AssignmentID {
		return nil, nil, ErrSubmissionForbidden
	}

	itemByID := make(map[string]domain.SubmissionItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}

	updates := make([]domain.SubmissionItem, 0, len(input.ItemScores))
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
		comment := &domain.SubmissionComment{
			ID:           uuid.NewString(),
			SubmissionID: submission.ID,
			AuthorID:     input.AccountID,
			AuthorRole:   domain.RoleTeacher,
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

// ReturnSubmissionInput captures return submission payload.
type ReturnSubmissionInput struct {
	AssignmentID string
	SubmissionID string
	AccountID    string
	Comment      string
}

// ReturnSubmission returns a submission to student for rework.
func (s *AssignmentService) ReturnSubmission(ctx context.Context, teacherID string, input ReturnSubmissionInput) (*SubmissionDetail, []domain.SubmissionComment, error) {
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

	// We use UpdateGrades here as it updates the submission record, even if items are empty
	if err := s.submissions.UpdateGrades(ctx, submission, []domain.SubmissionItem{}); err != nil {
		return nil, nil, err
	}

	if strings.TrimSpace(input.Comment) != "" {
		if input.AccountID == "" {
			return nil, nil, errors.New("account id required for comment")
		}
		comment := &domain.SubmissionComment{
			ID:           uuid.NewString(),
			SubmissionID: submission.ID,
			AuthorID:     input.AccountID,
			AuthorRole:   domain.RoleTeacher,
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

// UpdateAssignmentInput contains data for updating an assignment.
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

// UpdateAssignment updates an existing assignment.
func (s *AssignmentService) UpdateAssignment(ctx context.Context, input UpdateAssignmentInput) (*domain.Assignment, error) {
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

func mergeItems(original []domain.SubmissionItem, updates []domain.SubmissionItem) []domain.SubmissionItem {
	if len(updates) == 0 {
		return original
	}
	updated := make(map[string]domain.SubmissionItem, len(updates))
	for _, item := range updates {
		updated[item.ID] = item
	}
	result := make([]domain.SubmissionItem, 0, len(original))
	for _, item := range original {
		if upd, ok := updated[item.ID]; ok {
			result = append(result, upd)
		} else {
			result = append(result, item)
		}
	}
	return result
}
