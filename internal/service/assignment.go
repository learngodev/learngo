package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

// AssignmentService manages assignments and submissions.
type AssignmentService struct {
	assignments repository.AssignmentRepository
	submissions repository.SubmissionRepository
	comments    repository.SubmissionCommentRepository
}

// NewAssignmentService creates a new AssignmentService.
func NewAssignmentService(assignments repository.AssignmentRepository, submissions repository.SubmissionRepository, comments repository.SubmissionCommentRepository) *AssignmentService {
	return &AssignmentService{
		assignments: assignments,
		submissions: submissions,
		comments:    comments,
	}
}

// ErrAssignmentNotFound indicates the assignment does not exist.
var ErrAssignmentNotFound = errors.New("assignment not found")

// ErrSubmissionNotFound indicates the submission does not exist.
var ErrSubmissionNotFound = errors.New("submission not found")

// ErrSubmissionForbidden indicates the caller cannot access the submission.
var ErrSubmissionForbidden = errors.New("submission forbidden")

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

	if err := s.assignments.Create(ctx, assignment, questions); err != nil {
		return nil, err
	}
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

	submission := &domain.AssignmentSubmission{
		ID:           uuid.NewString(),
		AssignmentID: input.AssignmentID,
		StudentID:    input.StudentID,
		Score:        input.Score,
		Feedback:     input.Feedback,
		Status:       input.Status,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if input.Status == "submitted" {
		now := time.Now()
		submission.SubmittedAt = &now
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
	Submission domain.AssignmentSubmission
	Items      []domain.SubmissionItem
}

// GetAssignment retrieves an assignment with its questions.
func (s *AssignmentService) GetAssignment(ctx context.Context, assignmentID string) (*domain.Assignment, []domain.AssignmentQuestion, error) {
	if assignmentID == "" {
		return nil, nil, errors.New("assignment id required")
	}
	assignment, questions, err := s.assignments.Get(ctx, assignmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrAssignmentNotFound
		}
		return nil, nil, err
	}
	return assignment, questions, nil
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
	for _, sub := range subs {
		ids = append(ids, sub.ID)
	}

	items, err := s.submissions.ListItemsBySubmissionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	itemsBySubmission := make(map[string][]domain.SubmissionItem, len(subs))
	for _, item := range items {
		itemsBySubmission[item.SubmissionID] = append(itemsBySubmission[item.SubmissionID], item)
	}

	details := make([]SubmissionDetail, 0, len(subs))
	for _, sub := range subs {
		details = append(details, SubmissionDetail{
			Submission: sub,
			Items:      itemsBySubmission[sub.ID],
		})
	}
	return details, nil
}

// GetSubmissionForStudent returns the student's submission with items and comments.
func (s *AssignmentService) GetSubmissionForStudent(ctx context.Context, assignmentID, studentID string) (*SubmissionDetail, []domain.SubmissionComment, error) {
	if assignmentID == "" || studentID == "" {
		return nil, nil, errors.New("assignment and student required")
	}

	if _, _, err := s.GetAssignment(ctx, assignmentID); err != nil {
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

	assignment, _, err := s.GetAssignment(ctx, assignmentID)
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

	assignment, _, err := s.GetAssignment(ctx, input.AssignmentID)
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
