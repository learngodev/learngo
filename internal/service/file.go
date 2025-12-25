package service

import (
	"context"
	"fmt"
	"time"

	"learn-go/internal/domain"
	"learn-go/internal/repository"
	"learn-go/pkg/oss"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileService struct {
	db    *gorm.DB
	creds repository.OssCredentialRepository
}

func NewFileService(db *gorm.DB, creds repository.OssCredentialRepository) *FileService {
	return &FileService{
		db:    db,
		creds: creds,
	}
}

func (s *FileService) getClient(ctx context.Context, schoolID string) (oss.Client, error) {
	cred, err := s.creds.GetPrimary(ctx, schoolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get oss credential: %w", err)
	}
	return oss.NewAliyunClient(cred.Endpoint, cred.AccessKeyID, cred.AccessKeySecret, cred.Bucket)
}

// GetUploadURL generates a presigned URL for uploading a file.
func (s *FileService) GetUploadURL(ctx context.Context, schoolID, uploaderID, fileName, fileType string, size int64) (*domain.File, string, error) {
	client, err := s.getClient(ctx, schoolID)
	if err != nil {
		return nil, "", err
	}

	fileID := uuid.New().String()
	// Organize files by school/uploader/date/file_id
	key := fmt.Sprintf("%s/%s/%s/%s", schoolID, uploaderID, time.Now().Format("20060102"), fileID)

	// Generate presigned URL for PUT
	url, err := client.SignURL(key, "PUT", 3600) // 1 hour expiration
	if err != nil {
		return nil, "", err
	}

	// Create file record (status pending initially, but for simplicity we just create it)
	file := &domain.File{
		ID:         fileID,
		SchoolID:   schoolID,
		UploaderID: uploaderID,
		Name:       fileName,
		Key:        key,
		URL:        "", // Will be set after upload or generated on demand
		Type:       fileType,
		Size:       size,
		CreatedAt:  time.Now(),
	}

	if err := s.db.Create(file).Error; err != nil {
		return nil, "", err
	}

	return file, url, nil
}

// GetDownloadURL generates a presigned URL for downloading/viewing a file.
func (s *FileService) GetDownloadURL(ctx context.Context, fileID string) (string, error) {
	var file domain.File
	if err := s.db.First(&file, "id = ?", fileID).Error; err != nil {
		return "", err
	}

	client, err := s.getClient(ctx, file.SchoolID)
	if err != nil {
		return "", err
	}

	return client.SignURL(file.Key, "GET", 3600)
}

// AttachToAssignment links a file to an assignment.
func (s *FileService) AttachToAssignment(ctx context.Context, assignmentID, fileID string) error {
	attachment := &domain.AssignmentAttachment{
		ID:           uuid.New().String(),
		AssignmentID: assignmentID,
		FileID:       fileID,
		CreatedAt:    time.Now(),
	}
	return s.db.Create(attachment).Error
}

// AttachToSubmission links a file to a submission.
func (s *FileService) AttachToSubmission(ctx context.Context, submissionID, fileID string) error {
	attachment := &domain.SubmissionAttachment{
		ID:           uuid.New().String(),
		SubmissionID: submissionID,
		FileID:       fileID,
		CreatedAt:    time.Now(),
	}
	return s.db.Create(attachment).Error
}
