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

func TestTeacherPortalService_GetAssignmentDetail(t *testing.T) {
	teacherRepo := &fakeTeacherRepo{
		byAccount: map[string]*domain.Teacher{
			"acc-1": {ID: "teacher-1", AccountID: "acc-1", SchoolID: "school-1"},
		},
	}

	assignment := &domain.Assignment{
		ID:            "assign-1",
		CourseID:      "course-1",
		TeacherID:     "teacher-1",
		ClassID:       "class-1",
		Type:          domain.AssignmentHomework,
		Title:         "Chapter 1",
		Description:   "Do exercises",
		AllowResubmit: true,
		MaxScore:      100,
	}
	now := time.Now()
	assignment.StartAt = &now
	assignment.DueAt = &now

	questions := []domain.AssignmentQuestion{
		{ID: "q1", AssignmentID: assignment.ID, Type: domain.QuestionEssay, Prompt: "Q1", Options: "[]", Answer: "A", Score: 60, OrderIndex: 1},
		{ID: "q2", AssignmentID: assignment.ID, Type: domain.QuestionEssay, Prompt: "Q2", Options: "[]", Answer: "B", Score: 40, OrderIndex: 2},
	}

	assignmentRepo := &fakeAssignmentRepo{assignment: assignment, questions: questions}
	submissionRepo := &fakeSubmissionRepo{
		stats: map[string]repository.AssignmentSubmissionStats{
			"assign-1": {
				AssignmentID:      "assign-1",
				Total:             30,
				Submitted:         20,
				Graded:            10,
				AverageScore:      floatPtr(82.5),
				MaxScore:          floatPtr(98),
				MinScore:          floatPtr(60),
				LatestSubmittedAt: &now,
			},
		},
	}
	studentRepo := &fakeStudentRepo{counts: map[string]int64{"class-1": 32}}
	courseRepo := &fakeCourseRepo{courses: map[string]string{"course-1": "Math"}}
	classRepo := &fakeClassRepo{classes: map[string]string{"class-1": "Class One"}}

	svc := NewTeacherPortalService(
		teacherRepo,
		assignmentRepo,
		submissionRepo,
		studentRepo,
		nil,
		courseRepo,
		classRepo,
		nil,
		nil,
		nil,
	)

	detail, err := svc.GetAssignmentDetail(context.Background(), "acc-1", "assign-1", true)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if detail.CourseName != "Math" || detail.ClassName != "Class One" {
		t.Fatalf("unexpected names: %+v", detail)
	}
	if detail.Stats.SubmissionCount != 30 || detail.Stats.MissingCount != 12 {
		t.Fatalf("unexpected stats: %+v", detail.Stats)
	}
	if len(detail.Questions) != 2 || detail.Questions[0].ID != "q1" {
		t.Fatalf("unexpected questions: %+v", detail.Questions)
	}
}

func TestTeacherPortalService_GetAssignmentDetailForbidden(t *testing.T) {
	teacherRepo := &fakeTeacherRepo{
		byAccount: map[string]*domain.Teacher{
			"acc-1": {ID: "teacher-1", AccountID: "acc-1"},
		},
	}
	assignmentRepo := &fakeAssignmentRepo{assignment: &domain.Assignment{ID: "assign-1", TeacherID: "teacher-2", CourseID: "course-1", ClassID: "class-1"}}
	submissionRepo := &fakeSubmissionRepo{}
	studentRepo := &fakeStudentRepo{}
	courseRepo := &fakeCourseRepo{}
	classRepo := &fakeClassRepo{}

	svc := NewTeacherPortalService(teacherRepo, assignmentRepo, submissionRepo, studentRepo, nil, courseRepo, classRepo, nil, nil, nil)

	_, err := svc.GetAssignmentDetail(context.Background(), "acc-1", "assign-1", false)
	if !errors.Is(err, ErrTeacherAssignmentForbidden) {
		t.Fatalf("expected ErrTeacherAssignmentForbidden, got %v", err)
	}
}

func TestTeacherPortalService_GetAssignmentDetailHideAnswers(t *testing.T) {
	teacherRepo := &fakeTeacherRepo{
		byAccount: map[string]*domain.Teacher{
			"acc-1": {ID: "teacher-1", AccountID: "acc-1"},
		},
	}
	questions := []domain.AssignmentQuestion{
		{ID: "q1", AssignmentID: "assign-1", Type: domain.QuestionEssay, Prompt: "Q1", Options: "[]", Answer: "A", Score: 10, OrderIndex: 1},
	}
	assignmentRepo := &fakeAssignmentRepo{assignment: &domain.Assignment{ID: "assign-1", TeacherID: "teacher-1", CourseID: "course-1", ClassID: "class-1"}, questions: questions}
	submissionRepo := &fakeSubmissionRepo{stats: map[string]repository.AssignmentSubmissionStats{"assign-1": {AssignmentID: "assign-1"}}}
	studentRepo := &fakeStudentRepo{}
	courseRepo := &fakeCourseRepo{}
	classRepo := &fakeClassRepo{}
	svc := NewTeacherPortalService(teacherRepo, assignmentRepo, submissionRepo, studentRepo, nil, courseRepo, classRepo, nil, nil, nil)

	detail, err := svc.GetAssignmentDetail(context.Background(), "acc-1", "assign-1", false)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(detail.Questions) != 1 {
		t.Fatalf("unexpected question length")
	}
	if detail.Questions[0].Answer != "" || detail.Questions[0].Options != "" {
		t.Fatalf("expected hidden fields, got %+v", detail.Questions[0])
	}
}

func TestTeacherPortalService_ExportAssignmentGrades(t *testing.T) {
	teacherRepo := &fakeTeacherRepo{
		byAccount: map[string]*domain.Teacher{
			"acc-1": {ID: "teacher-1", AccountID: "acc-1"},
		},
	}
	assignment := &domain.Assignment{ID: "assign-1", TeacherID: "teacher-1", CourseID: "course-1", ClassID: "class-1", Title: "期末考试", Type: domain.AssignmentExam, MaxScore: 100}
	assignmentRepo := &fakeAssignmentRepo{assignment: assignment}
	scoreA := 95.0
	scoreB := 78.5
	firstSubmitted := time.Now()
	secondSubmitted := firstSubmitted.Add(-time.Hour)
	submissionRepo := &fakeSubmissionRepo{
		submissions: map[string][]domain.AssignmentSubmission{
			"assign-1": {
				{ID: "sub-1", AssignmentID: "assign-1", StudentID: "stu-1", Status: "graded", Score: &scoreA, SubmittedAt: &firstSubmitted},
				{ID: "sub-2", AssignmentID: "assign-1", StudentID: "stu-2", Status: "submitted", Score: &scoreB, SubmittedAt: &secondSubmitted},
			},
		},
	}
	studentRepo := &fakeStudentRepo{
		items: map[string]domain.Student{
			"stu-1": {ID: "stu-1", Number: "S001", AccountID: "acct-1"},
			"stu-2": {ID: "stu-2", Number: "S002", AccountID: "acct-2"},
		},
	}
	courseRepo := &fakeCourseRepo{courses: map[string]string{"course-1": "Math"}}
	classRepo := &fakeClassRepo{classes: map[string]string{"class-1": "Class One"}}
	accountRepo := &teacherAccountRepoStub{accounts: map[string]domain.Account{
		"acct-1": {ID: "acct-1", DisplayName: "Alice"},
		"acct-2": {ID: "acct-2", DisplayName: "Bob"},
	}}

	svc := NewTeacherPortalService(teacherRepo, assignmentRepo, submissionRepo, studentRepo, nil, courseRepo, classRepo, nil, accountRepo, nil)

	export, err := svc.ExportAssignmentGrades(context.Background(), "acc-1", "assign-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if export.ClassName != "Class One" || export.CourseName != "Math" {
		t.Fatalf("unexpected metadata: %+v", export)
	}
	if len(export.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(export.Rows))
	}
	if export.Rows[0].StudentNumber != "S001" || export.Rows[0].StudentName != "Alice" {
		t.Fatalf("unexpected first row: %+v", export.Rows[0])
	}
	if export.Rows[1].StudentNumber != "S002" || export.Rows[1].StudentName != "Bob" {
		t.Fatalf("unexpected second row: %+v", export.Rows[1])
	}
	if export.Rows[0].Score == nil || *export.Rows[0].Score != scoreA {
		t.Fatalf("unexpected score in first row: %+v", export.Rows[0].Score)
	}
	if export.Rows[1].Score == nil || *export.Rows[1].Score != scoreB {
		t.Fatalf("unexpected score in second row: %+v", export.Rows[1].Score)
	}
}

func floatPtr(v float64) *float64 { return &v }

type fakeTeacherRepo struct {
	byAccount map[string]*domain.Teacher
}

func (f *fakeTeacherRepo) Create(context.Context, *domain.Teacher) error { return nil }
func (f *fakeTeacherRepo) GetByNumber(context.Context, string, string) (*domain.Teacher, error) {
	return nil, nil
}
func (f *fakeTeacherRepo) GetByID(context.Context, string) (*domain.Teacher, error) { return nil, nil }
func (f *fakeTeacherRepo) GetByAccountID(ctx context.Context, accountID string) (*domain.Teacher, error) {
	if t, ok := f.byAccount[accountID]; ok {
		return t, nil
	}
	return nil, nil
}
func (f *fakeTeacherRepo) Update(context.Context, *domain.Teacher) error { return nil }

type fakeAssignmentRepo struct {
	assignment *domain.Assignment
	questions  []domain.AssignmentQuestion
}

func (f *fakeAssignmentRepo) Create(context.Context, *domain.Assignment, []domain.AssignmentQuestion, []domain.AssignmentAttachment) error {
	return nil
}
func (f *fakeAssignmentRepo) Get(ctx context.Context, id string) (*domain.Assignment, []domain.AssignmentQuestion, []domain.File, error) {
	if f.assignment != nil && f.assignment.ID == id {
		return f.assignment, f.questions, nil, nil
	}
	return nil, nil, nil, gorm.ErrRecordNotFound
}
func (f *fakeAssignmentRepo) ListByClass(context.Context, string, int, []domain.AssignmentType) ([]domain.Assignment, error) {
	return nil, nil
}
func (f *fakeAssignmentRepo) ListDueBetween(context.Context, string, time.Time, time.Time, []domain.AssignmentType) ([]domain.Assignment, error) {
	return nil, nil
}
func (f *fakeAssignmentRepo) ListByTeacher(context.Context, string, int, string, []domain.AssignmentType) ([]domain.Assignment, error) {
	return nil, nil
}
func (f *fakeAssignmentRepo) ListDueBetweenByTeacher(context.Context, string, time.Time, time.Time, []domain.AssignmentType) ([]domain.Assignment, error) {
	return nil, nil
}
func (f *fakeAssignmentRepo) Search(ctx context.Context, teacherID string, q string) ([]domain.Assignment, error) {
	return nil, nil
}
func (f *fakeAssignmentRepo) Update(ctx context.Context, assignment *domain.Assignment) error {
	f.assignment = assignment
	return nil
}

type fakeSubmissionRepo struct {
	stats       map[string]repository.AssignmentSubmissionStats
	submissions map[string][]domain.AssignmentSubmission
}

func (f *fakeSubmissionRepo) CreateOrUpdate(context.Context, *domain.AssignmentSubmission, []domain.SubmissionItem) error {
	return nil
}
func (f *fakeSubmissionRepo) ListByAssignment(ctx context.Context, assignmentID string) ([]domain.AssignmentSubmission, error) {
	if f.submissions == nil {
		return []domain.AssignmentSubmission{}, nil
	}
	subs := f.submissions[assignmentID]
	result := make([]domain.AssignmentSubmission, len(subs))
	copy(result, subs)
	return result, nil
}
func (f *fakeSubmissionRepo) ListItemsBySubmissionIDs(context.Context, []string) ([]domain.SubmissionItem, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) GetByAssignmentAndStudent(context.Context, string, string) (*domain.AssignmentSubmission, []domain.SubmissionItem, error) {
	return nil, nil, gorm.ErrRecordNotFound
}
func (f *fakeSubmissionRepo) GetByID(context.Context, string) (*domain.AssignmentSubmission, []domain.SubmissionItem, error) {
	return nil, nil, gorm.ErrRecordNotFound
}
func (f *fakeSubmissionRepo) UpdateGrades(context.Context, *domain.AssignmentSubmission, []domain.SubmissionItem) error {
	return nil
}
func (f *fakeSubmissionRepo) ListByStudentAndAssignments(context.Context, string, []string) ([]domain.AssignmentSubmission, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) StatsByAssignments(ctx context.Context, assignmentIDs []string) ([]repository.AssignmentSubmissionStats, error) {
	result := make([]repository.AssignmentSubmissionStats, 0, len(assignmentIDs))
	for _, id := range assignmentIDs {
		if stat, ok := f.stats[id]; ok {
			result = append(result, stat)
		}
	}
	return result, nil
}

type fakeStudentRepo struct {
	counts map[string]int64
	items  map[string]domain.Student
}

func (f *fakeStudentRepo) Create(context.Context, *domain.Student) error { return nil }
func (f *fakeStudentRepo) GetByNumber(context.Context, string, string) (*domain.Student, error) {
	return nil, nil
}
func (f *fakeStudentRepo) GetByID(context.Context, string) (*domain.Student, error) { return nil, nil }
func (f *fakeStudentRepo) GetByAccountID(context.Context, string) (*domain.Student, error) {
	return nil, nil
}
func (f *fakeStudentRepo) ListByIDs(ctx context.Context, ids []string) ([]domain.Student, error) {
	if len(ids) == 0 {
		return []domain.Student{}, nil
	}
	result := make([]domain.Student, 0, len(ids))
	for _, id := range ids {
		if f.items != nil {
			if student, ok := f.items[id]; ok {
				result = append(result, student)
			}
		}
	}
	return result, nil
}
func (f *fakeStudentRepo) CountByClassIDs(ctx context.Context, classIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	for _, id := range classIDs {
		if c, ok := f.counts[id]; ok {
			result[id] = c
		}
	}
	return result, nil
}
func (f *fakeStudentRepo) UpdateClassID(context.Context, string, string) error { return nil }
func (f *fakeStudentRepo) ListByClassID(context.Context, string) ([]domain.Student, error) {
	return nil, nil
}
func (f *fakeStudentRepo) ListByDepartmentID(context.Context, string) ([]domain.Student, error) {
	return nil, nil
}
func (f *fakeStudentRepo) Update(context.Context, *domain.Student) error { return nil }

type fakeCourseRepo struct {
	courses map[string]string
}

func (f *fakeCourseRepo) ListByIDs(ctx context.Context, ids []string) ([]domain.Course, error) {
	result := make([]domain.Course, 0, len(ids))
	for _, id := range ids {
		if name, ok := f.courses[id]; ok {
			result = append(result, domain.Course{ID: id, Name: name})
		}
	}
	return result, nil
}

func (f *fakeCourseRepo) Create(ctx context.Context, course *domain.Course) error { return nil }
func (f *fakeCourseRepo) List(ctx context.Context, schoolID string, page, size int) ([]domain.Course, int64, error) {
	return nil, 0, nil
}
func (f *fakeCourseRepo) GetByID(ctx context.Context, id string) (*domain.Course, error) {
	return nil, nil
}
func (f *fakeCourseRepo) Update(ctx context.Context, course *domain.Course) error { return nil }
func (f *fakeCourseRepo) Delete(ctx context.Context, id string) error             { return nil }
func (f *fakeCourseRepo) ListAssignments(ctx context.Context, schoolID string, courseID, departmentID, classID string, onlyAssigned bool, page, size int) ([]domain.CourseAssignmentInfo, int64, error) {
	return nil, 0, nil
}

func (f *fakeCourseRepo) ListWithFilters(ctx context.Context, schoolID string, departmentID, classID string, page, size int) ([]domain.Course, int64, error) {
	return nil, 0, nil
}

func (f *fakeCourseRepo) ListAssignmentsByCourseIDs(ctx context.Context, courseIDs []string) ([]domain.CourseAssignmentInfo, error) {
	return nil, nil
}

func (f *fakeCourseRepo) GetByInvitationCode(ctx context.Context, code string) (*domain.Course, error) {
	return nil, nil
}

func (f *fakeCourseRepo) ListByStudentID(ctx context.Context, studentID string) ([]domain.Course, error) {
	return nil, nil
}

type fakeClassRepo struct {
	classes map[string]string
}

func (f *fakeClassRepo) Create(context.Context, *domain.Class) error { return nil }
func (f *fakeClassRepo) ListByDepartment(context.Context, string, string) ([]domain.Class, error) {
	return nil, nil
}
func (f *fakeClassRepo) GetByID(context.Context, string) (*domain.Class, error) { return nil, nil }
func (f *fakeClassRepo) ListByIDs(ctx context.Context, ids []string) ([]domain.Class, error) {
	result := make([]domain.Class, 0, len(ids))
	for _, id := range ids {
		if name, ok := f.classes[id]; ok {
			result = append(result, domain.Class{ID: id, Name: name})
		}
	}
	return result, nil
}
func (f *fakeClassRepo) UpdateName(context.Context, string, string, string) error { return nil }
func (f *fakeClassRepo) Delete(context.Context, string, string) error             { return nil }
func (f *fakeClassRepo) AddTeacher(context.Context, string, string) error         { return nil }
func (f *fakeClassRepo) RemoveTeacher(context.Context, string, string) error      { return nil }

type teacherAccountRepoStub struct {
	accounts map[string]domain.Account
}

func (f *teacherAccountRepoStub) Create(context.Context, *domain.Account) error { return nil }
func (f *teacherAccountRepoStub) FindByIdentifier(context.Context, string, string) (*domain.Account, error) {
	return nil, nil
}
func (f *teacherAccountRepoStub) FindByID(context.Context, string) (*domain.Account, error) {
	return nil, nil
}
func (f *teacherAccountRepoStub) ListByIDs(ctx context.Context, ids []string) ([]domain.Account, error) {
	if len(ids) == 0 {
		return []domain.Account{}, nil
	}
	result := make([]domain.Account, 0, len(ids))
	for _, id := range ids {
		if account, ok := f.accounts[id]; ok {
			result = append(result, account)
		}
	}
	return result, nil
}
func (f *teacherAccountRepoStub) ListByRole(context.Context, string, domain.Role, domain.AccountStatus, string, string, string, bool, bool, int, int, string) ([]domain.Account, int64, error) {
	return nil, 0, nil
}
func (f *teacherAccountRepoStub) UpdateStatus(context.Context, string, string, domain.AccountStatus) error {
	return nil
}
func (f *teacherAccountRepoStub) UpdatePasswordHash(context.Context, string, string) error {
	return nil
}
func (f *teacherAccountRepoStub) Update(context.Context, *domain.Account) error { return nil }
func (f *teacherAccountRepoStub) Delete(context.Context, string, string) error  { return nil }
