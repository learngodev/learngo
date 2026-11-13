package usecase

import (
	"context"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/domain/curriculum"
	"learn-go/internal/domain/storage"

	"github.com/google/uuid"
)

func (s *TeacherPortalUsecase) ListCourseChapters(
	ctx context.Context,
	accountID string,
	courseID string,
) ([]curriculum.CourseChapter, error) {
	_, _, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return nil, err
	}

	return s.chapters.ListByCourse(ctx, courseID)
}

func (s *TeacherPortalUsecase) GetCourseChapter(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
) (*curriculum.CourseChapter, []storage.File, error) {
	_, _, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return nil, nil, err
	}

	chapter, err := s.chapters.GetByID(ctx, courseID, chapterID)
	if err != nil {
		return nil, nil, err
	}
	if chapter == nil {
		return nil, nil, domain.ErrNotFound
	}

	files, err := s.chapters.ListFiles(ctx, chapterID)
	if err != nil {
		return nil, nil, err
	}

	return chapter, files, nil
}

func (s *TeacherPortalUsecase) CreateCourseChapter(
	ctx context.Context,
	accountID string,
	courseID string,
	title string,
	content string,
	orderIndex int,
) (*curriculum.CourseChapter, error) {
	teacher, _, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	chapter := &curriculum.CourseChapter{
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

func (s *TeacherPortalUsecase) UpdateCourseChapter(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
	updates map[string]any,
) (*curriculum.CourseChapter, error) {
	_, _, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return nil, err
	}

	return s.chapters.UpdateFields(ctx, courseID, chapterID, updates)
}

func (s *TeacherPortalUsecase) DeleteCourseChapter(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
) error {
	_, _, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return err
	}

	// 确保章节属于该课程。
	chapter, err := s.chapters.GetByID(ctx, courseID, chapterID)
	if err != nil {
		return err
	}
	if chapter == nil {
		return domain.ErrNotFound
	}

	return s.chapters.Delete(ctx, courseID, chapterID)
}

func (s *TeacherPortalUsecase) AttachCourseChapterFile(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
	fileID string,
) error {
	_, _, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return err
	}

	chapter, err := s.chapters.GetByID(ctx, courseID, chapterID)
	if err != nil {
		return err
	}
	if chapter == nil {
		return domain.ErrNotFound
	}

	return s.chapters.AttachFile(ctx, chapterID, fileID)
}

func (s *TeacherPortalUsecase) DetachCourseChapterFile(
	ctx context.Context,
	accountID string,
	courseID string,
	chapterID string,
	fileID string,
) error {
	_, _, err := s.ensureTeacherCourseAccess(ctx, accountID, courseID)
	if err != nil {
		return err
	}

	chapter, err := s.chapters.GetByID(ctx, courseID, chapterID)
	if err != nil {
		return err
	}
	if chapter == nil {
		return domain.ErrNotFound
	}

	return s.chapters.DetachFile(ctx, chapterID, fileID)
}
