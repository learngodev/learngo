package usecase

import (
	"context"
	"errors"
	"learn-go/internal/domain"
	"math"
	"testing"
	"time"

	"learn-go/internal/domain/coursework"
	"learn-go/internal/domain/identity"
	"learn-go/internal/domain/storage"
)

type scoreValidationAssignmentRepo struct {
	assignment *coursework.Assignment
	questions  []coursework.AssignmentQuestion
}

func (f *scoreValidationAssignmentRepo) Create(context.Context, *coursework.Assignment, []coursework.AssignmentQuestion, []coursework.AssignmentAttachment) error {
	return nil
}

func (f *scoreValidationAssignmentRepo) Get(context.Context, string) (*coursework.Assignment, []coursework.AssignmentQuestion, []storage.File, error) {
	if f.assignment == nil {
		return nil, nil, nil, domain.ErrNotFound
	}
	return f.assignment, f.questions, nil, nil
}

func (f *scoreValidationAssignmentRepo) ListByClass(context.Context, string, int, []coursework.AssignmentType) ([]coursework.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) ListDueBetween(context.Context, string, time.Time, time.Time, []coursework.AssignmentType) ([]coursework.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) ListByTeacher(context.Context, string, int, string, []coursework.AssignmentType) ([]coursework.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) ListDueBetweenByTeacher(context.Context, string, time.Time, time.Time, []coursework.AssignmentType) ([]coursework.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) Search(context.Context, string, string) ([]coursework.Assignment, error) {
	return nil, nil
}

func (f *scoreValidationAssignmentRepo) Update(context.Context, *coursework.Assignment) error {
	return nil
}

type scoreValidationSubmissionRepo struct {
	submission           *coursework.AssignmentSubmission
	items                []coursework.SubmissionItem
	byAssignmentStudent  *coursework.AssignmentSubmission
	byAssignmentStudentI []coursework.SubmissionItem
	createdSubmission    *coursework.AssignmentSubmission
	createdItems         []coursework.SubmissionItem
	updatedSubmission    *coursework.AssignmentSubmission
	updatedItems         []coursework.SubmissionItem
}

func (f *scoreValidationSubmissionRepo) CreateOrUpdate(_ context.Context, submission *coursework.AssignmentSubmission, items []coursework.SubmissionItem) error {
	if submission != nil {
		copySubmission := *submission
		f.createdSubmission = &copySubmission
	}
	f.createdItems = append([]coursework.SubmissionItem(nil), items...)
	return nil
}

func (f *scoreValidationSubmissionRepo) ListByAssignment(context.Context, string) ([]coursework.AssignmentSubmission, error) {
	return nil, nil
}

func (f *scoreValidationSubmissionRepo) ListItemsBySubmissionIDs(context.Context, []string) ([]coursework.SubmissionItem, error) {
	return nil, nil
}

func (f *scoreValidationSubmissionRepo) GetByAssignmentAndStudent(context.Context, string, string) (*coursework.AssignmentSubmission, []coursework.SubmissionItem, error) {
	if f.byAssignmentStudent != nil {
		return f.byAssignmentStudent, f.byAssignmentStudentI, nil
	}
	return nil, nil, domain.ErrNotFound
}

func (f *scoreValidationSubmissionRepo) GetByID(context.Context, string) (*coursework.AssignmentSubmission, []coursework.SubmissionItem, error) {
	if f.submission == nil {
		return nil, nil, domain.ErrNotFound
	}
	return f.submission, f.items, nil
}

func (f *scoreValidationSubmissionRepo) UpdateGrades(_ context.Context, submission *coursework.AssignmentSubmission, items []coursework.SubmissionItem) error {
	if submission != nil {
		copySubmission := *submission
		f.updatedSubmission = &copySubmission
	}
	f.updatedItems = append([]coursework.SubmissionItem(nil), items...)
	return nil
}

func (f *scoreValidationSubmissionRepo) ListByStudentAndAssignments(context.Context, string, []string) ([]coursework.AssignmentSubmission, error) {
	return nil, nil
}

func (f *scoreValidationSubmissionRepo) StatsByAssignments(context.Context, []string) ([]coursework.AssignmentSubmissionStats, error) {
	return nil, nil
}

type scoreValidationCommentRepo struct {
	comments map[string][]coursework.SubmissionComment
}

func (f *scoreValidationCommentRepo) Create(_ context.Context, comment *coursework.SubmissionComment) error {
	if comment == nil {
		return nil
	}
	if f.comments == nil {
		f.comments = make(map[string][]coursework.SubmissionComment)
	}
	copyComment := *comment
	f.comments[comment.SubmissionID] = append(f.comments[comment.SubmissionID], copyComment)
	return nil
}

func (f *scoreValidationCommentRepo) ListBySubmission(_ context.Context, submissionID string) ([]coursework.SubmissionComment, error) {
	if f.comments == nil {
		return []coursework.SubmissionComment{}, nil
	}
	items := f.comments[submissionID]
	result := make([]coursework.SubmissionComment, len(items))
	copy(result, items)
	return result, nil
}

func TestCreateAssignmentRejectsWhenQuestionTotalExceedsMaxScore(t *testing.T) {
	svc := NewAssignmentUsecase(
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
			{Type: coursework.QuestionEssay, Prompt: "q1", Score: 60},
			{Type: coursework.QuestionEssay, Prompt: "q2", Score: 50},
		},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}

func TestCreateAssignmentRejectsWhenQuestionScoreExceedsAssignmentMax(t *testing.T) {
	svc := NewAssignmentUsecase(
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
		Questions: []QuestionInput{{Type: coursework.QuestionEssay, Prompt: "q1", Score: 120}},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}

func TestCreateAssignmentRejectsWhenMaxScoreNegative(t *testing.T) {
	svc := NewAssignmentUsecase(
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
		MaxScore:  -1,
		Questions: []QuestionInput{{Type: coursework.QuestionEssay, Prompt: "q1", Score: 1}},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}

func TestUpdateAssignmentRejectsWhenNewMaxScoreBelowQuestionTotal(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions: []coursework.AssignmentQuestion{
			{ID: "q1", Score: 40},
			{ID: "q2", Score: 40},
		},
	}
	svc := NewAssignmentUsecase(
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
		assignment: &coursework.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	svc := NewAssignmentUsecase(
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
		assignment: &coursework.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		byAssignmentStudent: &coursework.AssignmentSubmission{
			ID:           "sub-1",
			AssignmentID: "assign-1",
			StudentID:    "stu-1",
			Status:       "graded",
		},
	}
	svc := NewAssignmentUsecase(
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

func TestSubmitRejectsWhenItemScoreExceedsQuestionMax(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 10}},
	}
	svc := NewAssignmentUsecase(
		repo,
		&scoreValidationSubmissionRepo{},
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)
	tooHigh := 11.0
	err := svc.Submit(context.Background(), SubmitAssignmentInput{
		AssignmentID: "assign-1",
		StudentID:    "stu-1",
		Answers:      []AnswerInput{{QuestionID: "q1", Answer: "A", Score: &tooHigh}},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange, got %v", err)
	}
}

func TestSubmitRejectsWhenQuestionIDInvalid(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 10}},
	}
	svc := NewAssignmentUsecase(
		repo,
		&scoreValidationSubmissionRepo{},
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)
	one := 1.0
	err := svc.Submit(context.Background(), SubmitAssignmentInput{
		AssignmentID: "assign-1",
		StudentID:    "stu-1",
		Answers:      []AnswerInput{{QuestionID: "missing", Answer: "A", Score: &one}},
	})
	if err == nil || err.Error() != "invalid question id" {
		t.Fatalf("expected invalid question id error, got %v", err)
	}
}

func TestSubmitProgressCountsUniqueAnsweredQuestions(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", MaxScore: 100},
		questions: []coursework.AssignmentQuestion{
			{ID: "q1", Score: 50},
			{ID: "q2", Score: 50},
		},
	}
	subRepo := &scoreValidationSubmissionRepo{}
	svc := NewAssignmentUsecase(
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
		Answers: []AnswerInput{
			{QuestionID: "q1", Answer: "A"},
			{QuestionID: "q1", Answer: "A revised"},
			{QuestionID: "q2", Answer: "   "},
		},
	})
	if err != nil {
		t.Fatalf("expected submit to succeed, got %v", err)
	}
	if subRepo.createdSubmission == nil {
		t.Fatal("expected submission to be persisted")
	}
	if subRepo.createdSubmission.Progress != 50 {
		t.Fatalf("expected progress=50, got %d", subRepo.createdSubmission.Progress)
	}
}

func TestGradeSubmissionRejectsWhenItemScoreExceedsQuestionScore(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 20}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &coursework.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
		items:      []coursework.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
	}
	svc := NewAssignmentUsecase(
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
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &coursework.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
		items:      []coursework.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
	}
	svc := NewAssignmentUsecase(
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

func TestGetSubmissionForTeacherRejectsWhenTeacherDoesNotOwnAssignment(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1"},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &coursework.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
	}
	svc := NewAssignmentUsecase(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	_, _, err := svc.GetSubmissionForTeacher(context.Background(), "teacher-2", "assign-1", "sub-1")
	if !errors.Is(err, ErrSubmissionForbidden) {
		t.Fatalf("expected ErrSubmissionForbidden, got %v", err)
	}
}

func TestGradeSubmissionRejectsWhenTeacherDoesNotOwnAssignment(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 20}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &coursework.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
		items:      []coursework.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
	}
	svc := NewAssignmentUsecase(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	_, _, err := svc.GradeSubmission(context.Background(), "teacher-2", GradeSubmissionInput{
		AssignmentID: "assign-1",
		SubmissionID: "sub-1",
	})
	if !errors.Is(err, ErrSubmissionForbidden) {
		t.Fatalf("expected ErrSubmissionForbidden, got %v", err)
	}
}

func TestReturnSubmissionRejectsWhenTeacherDoesNotOwnAssignment(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1"},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &coursework.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
	}
	svc := NewAssignmentUsecase(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	_, _, err := svc.ReturnSubmission(context.Background(), "teacher-2", ReturnSubmissionInput{
		AssignmentID: "assign-1",
		SubmissionID: "sub-1",
	})
	if !errors.Is(err, ErrSubmissionForbidden) {
		t.Fatalf("expected ErrSubmissionForbidden, got %v", err)
	}
}

func TestReturnSubmissionUpdatesStatusAndCreatesComment(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1"},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &coursework.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1", Status: "submitted"},
		items:      []coursework.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
	}
	commentRepo := &scoreValidationCommentRepo{}
	svc := NewAssignmentUsecase(
		repo,
		subRepo,
		commentRepo,
		&fakeStudentRepo{},
		nil,
		nil,
	)

	_, comments, err := svc.ReturnSubmission(context.Background(), "teacher-1", ReturnSubmissionInput{
		AssignmentID: "assign-1",
		SubmissionID: "sub-1",
		AccountID:    "acc-1",
		Comment:      "please revise",
	})
	if err != nil {
		t.Fatalf("expected return submission to succeed, got %v", err)
	}
	if subRepo.updatedSubmission == nil {
		t.Fatal("expected submission update to be persisted")
	}
	if subRepo.updatedSubmission.Status != "returned" {
		t.Fatalf("expected status=returned, got %s", subRepo.updatedSubmission.Status)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].AuthorRole != identity.RoleTeacher {
		t.Fatalf("expected teacher author role, got %s", comments[0].AuthorRole)
	}
}

func TestCreateAssignmentRejectsWhenMaxScoreIsNaN(t *testing.T) {
	svc := NewAssignmentUsecase(
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
		MaxScore:  math.NaN(),
		Questions: []QuestionInput{{Type: coursework.QuestionEssay, Prompt: "q1", Score: 1}},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange for NaN max score, got %v", err)
	}
}

func TestCreateAssignmentRejectsWhenQuestionScoreIsPositiveInfinity(t *testing.T) {
	svc := NewAssignmentUsecase(
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
		Questions: []QuestionInput{{Type: coursework.QuestionEssay, Prompt: "q1", Score: math.Inf(1)}},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange for infinite question score, got %v", err)
	}
}

func TestUpdateAssignmentRejectsWhenNewMaxScoreIsNaN(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 20}},
	}
	svc := NewAssignmentUsecase(
		repo,
		&scoreValidationSubmissionRepo{},
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	v := math.NaN()
	_, err := svc.UpdateAssignment(context.Background(), UpdateAssignmentInput{
		ID:        "assign-1",
		TeacherID: "teacher-1",
		MaxScore:  &v,
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange for NaN update max score, got %v", err)
	}
}

func TestSubmitAllowsScoreEqualToAssignmentMax(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	subRepo := &scoreValidationSubmissionRepo{}
	svc := NewAssignmentUsecase(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	exact := 100.0
	err := svc.Submit(context.Background(), SubmitAssignmentInput{
		AssignmentID: "assign-1",
		StudentID:    "stu-1",
		Score:        &exact,
		Answers:      []AnswerInput{{QuestionID: "q1", Answer: "A"}},
	})
	if err != nil {
		t.Fatalf("expected exact max score submit to succeed, got %v", err)
	}
	if subRepo.createdSubmission == nil || subRepo.createdSubmission.Score == nil {
		t.Fatal("expected persisted submission score")
	}
	if *subRepo.createdSubmission.Score != exact {
		t.Fatalf("expected score %.2f, got %.2f", exact, *subRepo.createdSubmission.Score)
	}
}

func TestSubmitRejectsWhenScoreIsNaN(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	svc := NewAssignmentUsecase(
		repo,
		&scoreValidationSubmissionRepo{},
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	v := math.NaN()
	err := svc.Submit(context.Background(), SubmitAssignmentInput{
		AssignmentID: "assign-1",
		StudentID:    "stu-1",
		Score:        &v,
		Answers:      []AnswerInput{{QuestionID: "q1", Answer: "A"}},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange for NaN submission score, got %v", err)
	}
}

func TestSubmitRejectsWhenScoreIsNegative(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 100}},
	}
	svc := NewAssignmentUsecase(
		repo,
		&scoreValidationSubmissionRepo{},
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	v := -0.01
	err := svc.Submit(context.Background(), SubmitAssignmentInput{
		AssignmentID: "assign-1",
		StudentID:    "stu-1",
		Score:        &v,
		Answers:      []AnswerInput{{QuestionID: "q1", Answer: "A"}},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange for negative submission score, got %v", err)
	}
}

func TestGradeSubmissionAllowsExactUpperBounds(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 20}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &coursework.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1", Status: "submitted"},
		items:      []coursework.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
	}
	svc := NewAssignmentUsecase(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	total := 100.0
	item := 20.0
	_, _, err := svc.GradeSubmission(context.Background(), "teacher-1", GradeSubmissionInput{
		AssignmentID: "assign-1",
		SubmissionID: "sub-1",
		Score:        &total,
		ItemScores:   map[string]*float64{"item-1": &item},
	})
	if err != nil {
		t.Fatalf("expected grading with exact upper bounds to succeed, got %v", err)
	}
	if subRepo.updatedSubmission == nil || subRepo.updatedSubmission.Score == nil {
		t.Fatal("expected updated submission score to be persisted")
	}
	if *subRepo.updatedSubmission.Score != total {
		t.Fatalf("expected updated total score %.2f, got %.2f", total, *subRepo.updatedSubmission.Score)
	}
	if len(subRepo.updatedItems) != 1 || subRepo.updatedItems[0].Score == nil {
		t.Fatal("expected updated item score to be persisted")
	}
	if *subRepo.updatedItems[0].Score != item {
		t.Fatalf("expected updated item score %.2f, got %.2f", item, *subRepo.updatedItems[0].Score)
	}
}

func TestGradeSubmissionRejectsWhenItemScoreIsNaN(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 20}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &coursework.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
		items:      []coursework.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
	}
	svc := NewAssignmentUsecase(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	v := math.NaN()
	_, _, err := svc.GradeSubmission(context.Background(), "teacher-1", GradeSubmissionInput{
		AssignmentID: "assign-1",
		SubmissionID: "sub-1",
		ItemScores:   map[string]*float64{"item-1": &v},
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange for NaN item score, got %v", err)
	}
}

func TestGradeSubmissionRejectsWhenTotalScoreIsNegativeInfinity(t *testing.T) {
	repo := &scoreValidationAssignmentRepo{
		assignment: &coursework.Assignment{ID: "assign-1", TeacherID: "teacher-1", MaxScore: 100},
		questions:  []coursework.AssignmentQuestion{{ID: "q1", Score: 20}},
	}
	subRepo := &scoreValidationSubmissionRepo{
		submission: &coursework.AssignmentSubmission{ID: "sub-1", AssignmentID: "assign-1"},
		items:      []coursework.SubmissionItem{{ID: "item-1", QuestionID: "q1", SubmissionID: "sub-1"}},
	}
	svc := NewAssignmentUsecase(
		repo,
		subRepo,
		&scoreValidationCommentRepo{},
		&fakeStudentRepo{},
		nil,
		nil,
	)

	v := math.Inf(-1)
	_, _, err := svc.GradeSubmission(context.Background(), "teacher-1", GradeSubmissionInput{
		AssignmentID: "assign-1",
		SubmissionID: "sub-1",
		Score:        &v,
	})
	if !errors.Is(err, ErrScoreOutOfRange) {
		t.Fatalf("expected ErrScoreOutOfRange for infinite negative total score, got %v", err)
	}
}
