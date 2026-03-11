package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"gorm.io/gorm"
)

type scoreValidationAssignmentRepo struct {
	assignment *domain.Assignment
	questions  []domain.AssignmentQuestion
}

func (f *scoreValidationAssignmentRepo) Create(context.Context, *domain.Assignment, []domain.AssignmentQuestion, []domain.AssignmentAttachment) error {
	return nil
}

func (f *scoreValidationAssignmentRepo) Get(context.Context, string) (*domain.Assignment, []domain.AssignmentQuestion, []domain.File, error) {
	if f.assignment == nil {
		return nil, nil, nil, gorm.ErrRecordNotFound
	}
	return f.assignment, f.questions, nil, nil
}

func (f *scoreValidationAssignmentRepo) ListByClass(context.Context, string, int, []domain.AssignmentType) ([]domain.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) ListDueBetween(context.Context, string, time.Time, time.Time, []domain.AssignmentType) ([]domain.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) ListByTeacher(context.Context, string, int, string, []domain.AssignmentType) ([]domain.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) ListDueBetweenByTeacher(context.Context, string, time.Time, time.Time, []domain.AssignmentType) ([]domain.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) Search(context.Context, string, string) ([]domain.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) Update(context.Context, *domain.Assignment) error {
	return nil
}

type scoreValidationSubmissionRepo struct {
	submission *domain.AssignmentSubmission
	items      []domain.SubmissionItem
}

func (f *scoreValidationSubmissionRepo) CreateOrUpdate(context.Context, *domain.AssignmentSubmission, []domain.SubmissionItem) error {
	return nil
}

func (f *scoreValidationSubmissionRepo) ListByAssignment(context.Context, string) ([]domain.AssignmentSubmission, error) {
	return nil, nil
}

func (f *scoreValidationSubmissionRepo) ListItemsBySubmissionIDs(context.Context, []string) ([]domain.SubmissionItem, error) {
	return nil, nil
}

func (f *scoreValidationSubmissionRepo) GetByAssignmentAndStudent(context.Context, string, string) (*domain.AssignmentSubmission, []domain.SubmissionItem, error) {
	return nil, nil, gorm.ErrRecordNotFound
}

func (f *scoreValidationSubmissionRepo) GetByID(context.Context, string) (*domain.AssignmentSubmission, []domain.SubmissionItem, error) {
	if f.submission == nil {
		return nil, nil, gorm.ErrRecordNotFound
	}
	return f.submission, f.items, nil
}

func (f *scoreValidationSubmissionRepo) UpdateGrades(context.Context, *domain.AssignmentSubmission, []domain.SubmissionItem) error {
	return nil
}

func (f *scoreValidationSubmissionRepo) ListByStudentAndAssignments(context.Context, string, []string) ([]domain.AssignmentSubmission, error) {
	return nil, nil
}

func (f *scoreValidationSubmissionRepo) StatsByAssignments(context.Context, []string) ([]repository.AssignmentSubmissionStats, error) {
	return nil, nil
}

type scoreValidationCommentRepo struct{}

func (f *scoreValidationCommentRepo) Create(context.Context, *domain.SubmissionComment) error {
	return nil
}

func (f *scoreValidationCommentRepo) ListBySubmission(context.Context, string) ([]domain.SubmissionComment, error) {
	return []domain.SubmissionComment{}, nil
}

func TestCreateAssignmentRejectsWhenQuestionTotalExceedsMaxScore(t *testing.T) {
	svc := NewAssignmentService(
		&scoreValidationAssignmentRepo{},
		&scoreValidationSubmissionRepo{},
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	_, err := svc.CreateAssignment(context.Background(), CreateAssignmentInput{
		CourseID:  "course-1",
		TeacherID: "teacher-1",
		ClassID:   "class-1",
		MaxScore:  100,
		Questions: []QuestionInput{
			{Type: domain.QuestionEssay, Prompt: "q1", Score: 60},
			{Type: domain.QuestionEssay, Prompt: "q2", Score: 50},
		},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}

func TestUpdateAssignmentRejectsWhenNewMaxScoreBelowQuestionTotal(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &domain.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions: []domain.AssignmentQuestion{
			{ID: "q1", Score: 40},
			{ID: "q2", Score: 40},
		},
	}
	svc := NewAssignmentService(
		repo,
		&scoreValidationSubmissionRepo{},
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)
	newMax := 70.0
	_, err := svc.UpdateAssignment(context.Background(), UpdateAssignmentInput{
		ID:        "assign-1",
		TeacherID: "teacher-1",
		MaxScore:  &newMax,
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}

func TestSubmitRejectsWhenScoreExceedsAssignmentMax(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &domain.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []domain.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	svc := NewAssignmentService(
		repo,
		&scoreValidationSubmissionRepo{},
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)
	tooHigh := 120.0
	err := svc.Submit(context.Background(), SubmitAssignmentInput{
		AssignmentID: "assign-1",
		StudentID:    "stu-1",
		Score:        &tooHigh,
		Answers:      []AnswerInput{{QuestionID: "q1", Answer: "A"}},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}

func TestGradeSubmissionRejectsWhenItemScoreExceedsQuestionScore(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &domain.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []domain.AssignmentQuestion{{ID: "q1", Score: 20}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &domain.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
		items:      []domain.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
	}
	svc := NewAssignmentService(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)
	tooHigh := 25.0
	_, _, err := svc.GradeSubmission(context.Background(), "teacher-1", GradeSubmissionInput{
		AssignmentID: "assign-1",
		SubmissionID: "sub-1",
		ItemScores:   map[string]*float64{"item-1": &tooHigh},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}

func TestGradeSubmissionRejectsWhenTotalScoreExceedsAssignmentMax(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &domain.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []domain.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &domain.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
		items:      []domain.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
	}
	svc := NewAssignmentService(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)
	tooHigh := 101.0
	_, _, err := svc.GradeSubmission(context.Background(), "teacher-1", GradeSubmissionInput{
		AssignmentID: "assign-1",
		SubmissionID: "sub-1",
		Score:        &tooHigh,
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}
