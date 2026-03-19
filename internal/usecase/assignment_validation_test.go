package service

import (
	"context"
	"errors"
	"gorm.io/gorm"
	assignmentbiz "learn-go/internal/biz/assignment"
	storagebiz "learn-go/internal/biz/storage"
	"testing"
	"time"
)

type scoreValidationAssignmentRepo struct {
	assignment *assignmentbiz.Assignment
	questions  []assignmentbiz.AssignmentQuestion
}

func (f *scoreValidationAssignmentRepo) Create(context.Context, *assignmentbiz.Assignment, []assignmentbiz.AssignmentQuestion, []assignmentbiz.AssignmentAttachment) error {
	return nil
}

func (f *scoreValidationAssignmentRepo) Get(context.Context, string) (*assignmentbiz.Assignment, []assignmentbiz.AssignmentQuestion, []storagebiz.File, error) {
	if f.assignment == nil {
		return nil, nil, nil, gorm.ErrRecordNotFound
	}
	return f.assignment, f.questions, nil, nil
}

func (f *scoreValidationAssignmentRepo) ListByClass(context.Context, string, int, []assignmentbiz.AssignmentType) ([]assignmentbiz.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) ListDueBetween(context.Context, string, time.Time, time.Time, []assignmentbiz.AssignmentType) ([]assignmentbiz.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) ListByTeacher(context.Context, string, int, string, []assignmentbiz.AssignmentType) ([]assignmentbiz.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) ListDueBetweenByTeacher(context.Context, string, time.Time, time.Time, []assignmentbiz.AssignmentType) ([]assignmentbiz.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) Search(context.Context, string, string) ([]assignmentbiz.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) Update(context.Context, *assignmentbiz.Assignment) error {
	return nil
}

type scoreValidationSubmissionRepo struct {
	submission           *assignmentbiz.AssignmentSubmission
	items                []assignmentbiz.SubmissionItem
	byAssignmentStudent  *assignmentbiz.AssignmentSubmission
	byAssignmentStudentI []assignmentbiz.SubmissionItem
}

func (f *scoreValidationSubmissionRepo) CreateOrUpdate(context.Context, *assignmentbiz.AssignmentSubmission, []assignmentbiz.SubmissionItem) error {
	return nil
}

func (f *scoreValidationSubmissionRepo) ListByAssignment(context.Context, string) ([]assignmentbiz.AssignmentSubmission, error) {
	return nil, nil
}

func (f *scoreValidationSubmissionRepo) ListItemsBySubmissionIDs(context.Context, []string) ([]assignmentbiz.SubmissionItem, error) {
	return nil, nil
}

func (f *scoreValidationSubmissionRepo) GetByAssignmentAndStudent(context.Context, string, string) (*assignmentbiz.AssignmentSubmission, []assignmentbiz.SubmissionItem, error) {
	if f.byAssignmentStudent != nil {
		return f.byAssignmentStudent, f.byAssignmentStudentI, nil
	}
	return nil, nil, gorm.ErrRecordNotFound
}

func (f *scoreValidationSubmissionRepo) GetByID(context.Context, string) (*assignmentbiz.AssignmentSubmission, []assignmentbiz.SubmissionItem, error) {
	if f.submission == nil {
		return nil, nil, gorm.ErrRecordNotFound
	}
	return f.submission, f.items, nil
}

func (f *scoreValidationSubmissionRepo) UpdateGrades(context.Context, *assignmentbiz.AssignmentSubmission, []assignmentbiz.SubmissionItem) error {
	return nil
}

func (f *scoreValidationSubmissionRepo) ListByStudentAndAssignments(context.Context, string, []string) ([]assignmentbiz.AssignmentSubmission, error) {
	return nil, nil
}

func (f *scoreValidationSubmissionRepo) StatsByAssignments(context.Context, []string) ([]assignmentbiz.AssignmentSubmissionStats, error) {
	return nil, nil
}

type scoreValidationCommentRepo struct{}

func (f *scoreValidationCommentRepo) Create(context.Context, *assignmentbiz.SubmissionComment) error {
	return nil
}

func (f *scoreValidationCommentRepo) ListBySubmission(context.Context, string) ([]assignmentbiz.SubmissionComment, error) {
	return []assignmentbiz.SubmissionComment{}, nil
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
			{Type: assignmentbiz.QuestionEssay, Prompt: "q1", Score: 60},
			{Type: assignmentbiz.QuestionEssay, Prompt: "q2", Score: 50},
		},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}

func TestUpdateAssignmentRejectsWhenNewMaxScoreBelowQuestionTotal(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &assignmentbiz.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions: []assignmentbiz.AssignmentQuestion{
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
		assignment: &assignmentbiz.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []assignmentbiz.AssignmentQuestion{{ID: "q1", Score: 100}},
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

func TestSubmitRejectsWhenExistingSubmissionAlreadyGraded(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &assignmentbiz.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []assignmentbiz.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		byAssignmentStudent: &assignmentbiz.AssignmentSubmission{
			ID:           "sub-1",
			AssignmentID: "assign-1",
			StudentID:    "stu-1",
			Status:       "graded",
		},
	}
	svc := NewAssignmentService(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	err := svc.Submit(context.Background(), SubmitAssignmentInput{
		AssignmentID: "assign-1",
		StudentID:    "stu-1",
		Answers:      []AnswerInput{{QuestionID: "q1", Answer: "updated answer"}},
		Status:       "submitted",
	})
	if !errors.Is(err, ErrSubmissionAlreadyGraded) {
		t.Fatalf("expected ErrSubmissionAlreadyGraded, got %v", err)
	}
}

func TestGradeSubmissionRejectsWhenItemScoreExceedsQuestionScore(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &assignmentbiz.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []assignmentbiz.AssignmentQuestion{{ID: "q1", Score: 20}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &assignmentbiz.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
		items:      []assignmentbiz.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
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
		assignment: &assignmentbiz.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []assignmentbiz.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &assignmentbiz.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
		items:      []assignmentbiz.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
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
