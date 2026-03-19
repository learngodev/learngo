package course

import (
	"context"

	"learn-go/internal/biz/storage"
)

// CourseChapterRepository manages learning chapters for a course.
type CourseChapterRepository interface {
	ListByCourse(ctx context.Context, courseID string) ([]CourseChapter, error)
	GetByID(ctx context.Context, courseID string, chapterID string) (*CourseChapter, error)
	ListFiles(ctx context.Context, chapterID string) ([]storage.File, error)
	Create(ctx context.Context, chapter *CourseChapter) error
	UpdateFields(ctx context.Context, courseID string, chapterID string, updates map[string]any) (*CourseChapter, error)
	Delete(ctx context.Context, courseID string, chapterID string) error
	AttachFile(ctx context.Context, chapterID string, fileID string) error
	DetachFile(ctx context.Context, chapterID string, fileID string) error
}
