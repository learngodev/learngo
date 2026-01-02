package gormrepo

import (
	"context"
	"learn-go/internal/domain"
	"learn-go/internal/repository"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CourseChapterStore implements repository.CourseChapterRepository.
type CourseChapterStore struct {
	db *gorm.DB
}

// NewCourseChapterStore constructs CourseChapterStore.
func NewCourseChapterStore(db *gorm.DB) *CourseChapterStore {
	return &CourseChapterStore{db: db}
}

func (s *CourseChapterStore) ListByCourse(ctx context.Context, courseID string) ([]domain.CourseChapter, error) {
	var chapters []domain.CourseChapter
	if err := s.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("order_index ASC, created_at ASC").
		Find(&chapters).Error; err != nil {
		return nil, err
	}
	return chapters, nil
}

func (s *CourseChapterStore) GetByID(ctx context.Context, courseID string, chapterID string) (*domain.CourseChapter, error) {
	var chapter domain.CourseChapter
	if err := s.db.WithContext(ctx).
		Where("id = ? AND course_id = ?", chapterID, courseID).
		First(&chapter).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &chapter, nil
}

func (s *CourseChapterStore) ListFiles(ctx context.Context, chapterID string) ([]domain.File, error) {
	var files []domain.File
	if err := s.db.WithContext(ctx).
		Table("files").
		Joins("JOIN course_chapter_attachments cca ON cca.file_id = files.id").
		Where("cca.chapter_id = ?", chapterID).
		Order("cca.created_at ASC").
		Find(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}

func (s *CourseChapterStore) Create(ctx context.Context, chapter *domain.CourseChapter) error {
	return s.db.WithContext(ctx).Create(chapter).Error
}

func (s *CourseChapterStore) UpdateFields(
	ctx context.Context,
	courseID string,
	chapterID string,
	updates map[string]any,
) (*domain.CourseChapter, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	updates["updated_at"] = time.Now()

	result := s.db.WithContext(ctx).
		Model(&domain.CourseChapter{}).
		Where("id = ? AND course_id = ?", chapterID, courseID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, repository.ErrNotFound
	}
	return s.GetByID(ctx, courseID, chapterID)
}

func (s *CourseChapterStore) Delete(ctx context.Context, courseID string, chapterID string) error {
	// Remove attachments first.
	if err := s.db.WithContext(ctx).
		Where("chapter_id = ?", chapterID).
		Delete(&domain.CourseChapterAttachment{}).Error; err != nil {
		return err
	}

	result := s.db.WithContext(ctx).
		Where("id = ? AND course_id = ?", chapterID, courseID).
		Delete(&domain.CourseChapter{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (s *CourseChapterStore) AttachFile(ctx context.Context, chapterID string, fileID string) error {
	// Ensure file exists.
	var count int64
	if err := s.db.WithContext(ctx).
		Model(&domain.File{}).
		Where("id = ?", fileID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return repository.ErrNotFound
	}

	// Avoid duplicate links.
	var existing int64
	if err := s.db.WithContext(ctx).
		Model(&domain.CourseChapterAttachment{}).
		Where("chapter_id = ? AND file_id = ?", chapterID, fileID).
		Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	link := domain.CourseChapterAttachment{
		ID:        uuid.NewString(),
		ChapterID: chapterID,
		FileID:    fileID,
		CreatedAt: time.Now(),
	}
	return s.db.WithContext(ctx).Create(&link).Error
}

func (s *CourseChapterStore) DetachFile(ctx context.Context, chapterID string, fileID string) error {
	return s.db.WithContext(ctx).
		Where("chapter_id = ? AND file_id = ?", chapterID, fileID).
		Delete(&domain.CourseChapterAttachment{}).Error
}
