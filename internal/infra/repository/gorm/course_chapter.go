package gormrepo

import (
	"context"
	"learn-go/internal/domain"
	"learn-go/internal/domain/curriculum"
	"learn-go/internal/domain/storage"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CourseChapterStore 实现 curriculum.CourseChapterRepository。
type CourseChapterStore struct {
	db *gorm.DB
}

// NewCourseChapterStore 构造 CourseChapterStore。
func NewCourseChapterStore(db *gorm.DB) *CourseChapterStore {
	return &CourseChapterStore{db: db}
}

func (s *CourseChapterStore) ListByCourse(ctx context.Context, courseID string) ([]curriculum.CourseChapter, error) {
	var chapters []curriculum.CourseChapter
	if err := s.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("order_index ASC, created_at ASC").
		Find(&chapters).Error; err != nil {
		return nil, err
	}
	return chapters, nil
}

func (s *CourseChapterStore) GetByID(ctx context.Context, courseID string, chapterID string) (*curriculum.CourseChapter, error) {
	var chapter curriculum.CourseChapter
	if err := s.db.WithContext(ctx).
		Where("id = ? AND course_id = ?", chapterID, courseID).
		First(&chapter).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &chapter, nil
}

func (s *CourseChapterStore) ListFiles(ctx context.Context, chapterID string) ([]storage.File, error) {
	var files []storage.File
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

func (s *CourseChapterStore) Create(ctx context.Context, chapter *curriculum.CourseChapter) error {
	return s.db.WithContext(ctx).Create(chapter).Error
}

func (s *CourseChapterStore) UpdateFields(
	ctx context.Context,
	courseID string,
	chapterID string,
	updates map[string]any,
) (*curriculum.CourseChapter, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	updates["updated_at"] = time.Now()

	result := s.db.WithContext(ctx).
		Model(&curriculum.CourseChapter{}).
		Where("id = ? AND course_id = ?", chapterID, courseID).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}
	return s.GetByID(ctx, courseID, chapterID)
}

func (s *CourseChapterStore) Delete(ctx context.Context, courseID string, chapterID string) error {
	// 先删除附件关联。
	if err := s.db.WithContext(ctx).
		Where("chapter_id = ?", chapterID).
		Delete(&curriculum.CourseChapterAttachment{}).Error; err != nil {
		return err
	}

	result := s.db.WithContext(ctx).
		Where("id = ? AND course_id = ?", chapterID, courseID).
		Delete(&curriculum.CourseChapter{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *CourseChapterStore) AttachFile(ctx context.Context, chapterID string, fileID string) error {
	// 确认文件存在。
	var count int64
	if err := s.db.WithContext(ctx).
		Model(&storage.File{}).
		Where("id = ?", fileID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrNotFound
	}

	// 避免重复关联。
	var existing int64
	if err := s.db.WithContext(ctx).
		Model(&curriculum.CourseChapterAttachment{}).
		Where("chapter_id = ? AND file_id = ?", chapterID, fileID).
		Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	link := curriculum.CourseChapterAttachment{
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
		Delete(&curriculum.CourseChapterAttachment{}).Error
}

var _ curriculum.CourseChapterRepository = (*CourseChapterStore)(nil)
