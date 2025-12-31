package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
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

func (s *FileService) getPrimaryCredential(ctx context.Context, schoolID string) (*domain.OssCredential, error) {
	cred, err := s.creds.GetPrimary(ctx, schoolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get oss credential: %w", err)
	}
	return cred, nil
}

func (s *FileService) newClient(cred *domain.OssCredential) (oss.Client, error) {
	return oss.NewAliyunClient(cred.Endpoint, cred.AccessKeyID, cred.AccessKeySecret, cred.Bucket)
}

func (s *FileService) newFileRecord(schoolID, uploaderID, fileName, fileType string, size int64) *domain.File {
	fileID := uuid.New().String()
	key := fmt.Sprintf("%s/%s/%s/%s", schoolID, uploaderID, time.Now().Format("20060102"), fileID)
	return &domain.File{
		ID:         fileID,
		SchoolID:   schoolID,
		UploaderID: uploaderID,
		Name:       fileName,
		Key:        key,
		URL:        "",
		Type:       fileType,
		Size:       size,
		CreatedAt:  time.Now(),
	}
}

// RelayUpload uploads file content to OSS through the server (no presigned URL).
// It streams the provided reader directly to OSS and then creates a DB record.
func (s *FileService) RelayUpload(ctx context.Context, schoolID, uploaderID, fileName, fileType string, size int64, reader io.Reader) (*domain.File, error) {
	cred, err := s.getPrimaryCredential(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	client, err := s.newClient(cred)
	if err != nil {
		return nil, err
	}

	file := s.newFileRecord(schoolID, uploaderID, fileName, fileType, size)
	if err := client.PutObject(file.Key, reader); err != nil {
		return nil, fmt.Errorf("failed to upload to oss: %w", err)
	}

	if err := s.db.WithContext(ctx).Create(file).Error; err != nil {
		_ = client.DeleteObject(file.Key) // best effort cleanup
		return nil, err
	}

	return file, nil
}

// GetFileForUploader returns a file record owned by a specific uploader in a school.
func (s *FileService) GetFileForUploader(ctx context.Context, schoolID, uploaderID, fileID string) (*domain.File, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file_id required")
	}
	var file domain.File
	if err := s.db.WithContext(ctx).First(&file, "id = ? AND school_id = ? AND uploader_id = ?", fileID, schoolID, uploaderID).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

// RelayUploadExisting uploads file content to OSS through the server using an existing files record.
// Caller should validate file metadata (name/type/size) before invoking this method.
func (s *FileService) RelayUploadExisting(ctx context.Context, file *domain.File, reader io.Reader) error {
	if file == nil {
		return errors.New("file required")
	}
	cred, err := s.getPrimaryCredential(ctx, file.SchoolID)
	if err != nil {
		return err
	}
	client, err := s.newClient(cred)
	if err != nil {
		return err
	}
	if err := client.PutObject(file.Key, reader); err != nil {
		return fmt.Errorf("failed to upload to oss: %w", err)
	}
	return nil
}

// InitUpload creates a files record and returns the upload method for the current primary OSS credential.
// - uploadMethod = "direct": returns a presigned PUT URL.
// - uploadMethod = "relay": client should POST multipart/form-data to server relay endpoint with file_id.
func (s *FileService) InitUpload(ctx context.Context, schoolID, uploaderID, fileName, fileType string, size int64) (*domain.File, string, string, error) {
	cred, err := s.getPrimaryCredential(ctx, schoolID)
	if err != nil {
		return nil, "", "", err
	}

	file := s.newFileRecord(schoolID, uploaderID, fileName, fileType, size)
	if err := s.db.WithContext(ctx).Create(file).Error; err != nil {
		return nil, "", "", err
	}

	if cred.UseRelayUpload {
		return file, "relay", "", nil
	}

	client, err := s.newClient(cred)
	if err != nil {
		return nil, "", "", err
	}

	url, err := client.SignURL(file.Key, "PUT", 3600)
	if err != nil {
		return nil, "", "", err
	}

	return file, "direct", url, nil
}

// GetUploadURL generates a presigned URL for uploading a file.
func (s *FileService) GetUploadURL(ctx context.Context, schoolID, uploaderID, fileName, fileType string, size int64) (*domain.File, string, error) {
	file, _, url, err := s.InitUpload(ctx, schoolID, uploaderID, fileName, fileType, size)
	if err != nil {
		return nil, "", err
	}
	return file, url, nil
}

// GetDownloadInfo returns the file record plus download method for the current primary OSS credential.
// - downloadMethod = "direct": returns a presigned GET URL.
// - downloadMethod = "relay": caller should use server relay download endpoint.
func (s *FileService) GetDownloadInfo(ctx context.Context, schoolID, fileID string) (*domain.File, string, string, error) {
	if strings.TrimSpace(fileID) == "" {
		return nil, "", "", fmt.Errorf("file id required")
	}

	var file domain.File
	if err := s.db.WithContext(ctx).First(&file, "id = ? AND school_id = ?", fileID, schoolID).Error; err != nil {
		return nil, "", "", err
	}

	cred, err := s.getPrimaryCredential(ctx, schoolID)
	if err != nil {
		return nil, "", "", err
	}
	if cred.UseRelayUpload {
		return &file, "relay", "", nil
	}

	client, err := s.newClient(cred)
	if err != nil {
		return nil, "", "", err
	}
	url, err := client.SignURL(file.Key, "GET", 3600)
	if err != nil {
		return nil, "", "", err
	}
	return &file, "direct", url, nil
}

// OpenDownloadStream opens an OSS object stream for server-side relay download.
func (s *FileService) OpenDownloadStream(ctx context.Context, schoolID, fileID string) (*domain.File, io.ReadCloser, error) {
	if strings.TrimSpace(fileID) == "" {
		return nil, nil, fmt.Errorf("file id required")
	}
	var file domain.File
	if err := s.db.WithContext(ctx).First(&file, "id = ? AND school_id = ?", fileID, schoolID).Error; err != nil {
		return nil, nil, err
	}

	cred, err := s.getPrimaryCredential(ctx, schoolID)
	if err != nil {
		return nil, nil, err
	}
	client, err := s.newClient(cred)
	if err != nil {
		return nil, nil, err
	}

	rc, err := client.GetObject(file.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get oss object: %w", err)
	}
	return &file, rc, nil
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
