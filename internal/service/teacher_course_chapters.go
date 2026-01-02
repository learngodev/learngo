package service

import (
	"context"
	"errors"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"

	"github.com/google/uuid"
)

// ErrTeacherCourseAccessDenied indicates the teacher doesn't have access to the course.
var ErrTeacherCourseAccessDenied = errors.New("teacher course access denied")

func (s *TeacherPortalService) ensureTeacherCourseAccess(
	ctx context.Context,
	accountID string,
	courseID string,
) (*domain.Teacher, error) {
	teacher, err := s.teachers.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if teacher == nil {
		return nil, ErrTeacherProfileNotFound
	}

	schedules, err := s.schedules.ListByTeacher(ctx, teacher.ID)
	if err != nil {
		return nil, err
	}

	for _, sch := range schedules {
		if sch.CourseID == courseID {
			return teacher, nil
		}
	}

	return nil, ErrTeacherCourseAccessDenied
}

func (s *TeacherPortalService) ListCourseChapters(
	ctx context.Context,
	accountID string,
	courseID string,
) ([]domain.CourseChapter, error) {
	_, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return nil, err
	}

	return s.chapters.ListByCourse(ctx, courseID)
}

func (s *TeacherPortalService) GetCourseChapter(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
) (*domain.CourseChapter, []domain.File, error) {
	_, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return nil, nil, err
	}

	chapter, err := s.chapters.GetByID(ctx, courseID, chapterID)
	if err != nil {
		return nil, nil, err
	}
	if chapter == nil {
		return nil, nil, repository.ErrNotFound
	}

	files, err := s.chapters.ListFiles(ctx, chapterID)
	if err != nil {
		return nil, nil, err
	}

	return chapter, files, nil
}

func (s *TeacherPortalService) CreateCourseChapter(
	ctx context.Context,
	accountID string,
	courseID string,
	title string,
	content string,
	orderIndex int,
) (*domain.CourseChapter, error) {
	teacher, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	chapter := &domain.CourseChapter{
		ID:         uuid.NewString(),
		CourseID:   courseID,
		TeacherID:  teacher.ID,
		Title:      title,
		Content:    content,
		OrderIndex: orderIndex,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.chapters.Create(ctx, chapter); err != nil {
		return nil, err
	}

	return chapter, nil
}

func (s *TeacherPortalService) UpdateCourseChapter(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
	updates map[string]any,
) (*domain.CourseChapter, error) {
	_, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return nil, err
	}

	return s.chapters.UpdateFields(ctx, courseID, chapterID, updates)
}

func (s *TeacherPortalService) DeleteCourseChapter(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
) error {
	_, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return err
	}

	// Ensure chapter belongs to the course.
	chapter, err := s.chapters.GetByID(ctx, courseID, chapterID)
	if err != nil {
		return err
	}
	if chapter == nil {
		return repository.ErrNotFound
	}

	return s.chapters.Delete(ctx, courseID, chapterID)
}

func (s *TeacherPortalService) AttachCourseChapterFile(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
	fileID string,
) error {
	_, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return err
	}

	chapter, err := s.chapters.GetByID(ctx, courseID, chapterID)
	if err != nil {
		return err
	}
	if chapter == nil {
		return repository.ErrNotFound
	}

	return s.chapters.AttachFile(ctx, chapterID, fileID)
}

func (s *TeacherPortalService) DetachCourseChapterFile(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
	fileID string,
) error {
	_, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return err
	}

	chapter, err := s.chapters.GetByID(ctx, courseID, chapterID)
	if err != nil {
		return err
	}
	if chapter == nil {
		return repository.ErrNotFound
	}

	return s.chapters.DetachFile(ctx, chapterID, fileID)
}
