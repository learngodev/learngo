package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"learn-go/internal/domain/coursework"
	"learn-go/internal/domain/storage"

	"github.com/google/uuid"
)

type FileUsecase struct {
	files       storage.FileRepository
	attachments coursework.AttachmentRepository
	creds       storage.OssCredentialRepository
	storage     storage.ObjectStorageFactory
}

func NewFileUsecase(
	files storage.FileRepository,
	attachments coursework.AttachmentRepository,
	creds storage.OssCredentialRepository,
	objectStorage storage.ObjectStorageFactory,
) *FileUsecase {
	return &FileUsecase{
		files:       files,
		attachments: attachments,
		creds:       creds,
		storage:     objectStorage,
	}
}

func (s *FileUsecase) getPrimaryCredential(ctx context.Context, schoolID string) (*storage.OssCredential, error) {
	cred, err := s.creds.GetPrimary(ctx, schoolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get oss credential: %w", err)
	}
	return cred, nil
}

func (s *FileUsecase) newClient(cred *storage.OssCredential) (storage.ObjectStorage, error) {
	return s.newClientWithEndpoint(cred, s.uploadEndpoint(cred))
}

func (s *FileUsecase) uploadEndpoint(cred *storage.OssCredential) string {
	if cred == nil {
		return ""
	}
	if ep := strings.TrimSpace(cred.InternalEndpoint); ep != "" {
		return ep
	}
	return strings.TrimSpace(cred.Endpoint)
}

func (s *FileUsecase) downloadEndpoint(cred *storage.OssCredential) string {
	if cred == nil {
		return ""
	}
	// 返回给客户端的下载地址应始终使用公网 endpoint。
	ep := strings.TrimSpace(cred.Endpoint)
	if strings.TrimSpace(cred.InternalEndpoint) == "" && strings.Contains(ep, "-internal.") {
		// 阿里云 OSS 常见格式：oss-<region>-internal.aliyuncs.com -> oss-<region>.aliyuncs.com
		return strings.Replace(ep, "-internal.", ".", 1)
	}
	return ep
}

func (s *FileUsecase) newClientWithEndpoint(cred *storage.OssCredential, endpoint string) (storage.ObjectStorage, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("oss endpoint required")
	}
	return s.storage.NewClient(endpoint, cred)
}

func (s *FileUsecase) newFileRecord(schoolID, uploaderID, fileName, fileType string, size int64) *storage.File {
	fileID := uuid.New().String()
	key := fmt.Sprintf("%s/%s/%s/%s", schoolID, uploaderID, time.Now().Format("20060102"), fileID)
	return &storage.File{
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

// RelayUpload 由服务端中转将文件内容上传到 OSS（不使用预签名 URL）。
// 将 reader 内容直接流式写入 OSS，然后创建数据库记录。
func (s *FileUsecase) RelayUpload(ctx context.Context, schoolID, uploaderID, fileName, fileType string, size int64, reader io.Reader) (*storage.File, error) {
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

	if err := s.files.Create(ctx, file); err != nil {
		_ = client.DeleteObject(file.Key) // 尽力清理
		return nil, err
	}

	return file, nil
}

// GetFileForUploader 返回某学校中指定上传者拥有的文件记录。
func (s *FileUsecase) GetFileForUploader(ctx context.Context, schoolID, uploaderID, fileID string) (*storage.File, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file_id required")
	}
	return s.files.GetByUploader(ctx, fileID, schoolID, uploaderID)
}

// RelayUploadExisting 基于已有的文件记录，由服务端中转将文件内容上传到 OSS。
// 调用方应在调用前校验文件元数据（名称/类型/大小）。
func (s *FileUsecase) RelayUploadExisting(ctx context.Context, file *storage.File, reader io.Reader) error {
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

// InitUpload 创建文件记录，并根据当前主 OSS 凭据返回上传方式。
// - uploadMethod = "direct"：返回预签名 PUT URL。
// - uploadMethod = "relay"：客户端应携带 file_id，以 multipart/form-data 方式 POST 到服务端中转接口。
func (s *FileUsecase) InitUpload(ctx context.Context, schoolID, uploaderID, fileName, fileType string, size int64) (*storage.File, string, string, error) {
	cred, err := s.getPrimaryCredential(ctx, schoolID)
	if err != nil {
		return nil, "", "", err
	}

	file := s.newFileRecord(schoolID, uploaderID, fileName, fileType, size)
	if err := s.files.Create(ctx, file); err != nil {
		return nil, "", "", err
	}

	if cred.UseRelayUpload {
		return file, "relay", "", nil
	}

	client, err := s.newClientWithEndpoint(cred, s.uploadEndpoint(cred))
	if err != nil {
		return nil, "", "", err
	}

	url, err := client.SignURL(file.Key, "PUT", 3600)
	if err != nil {
		return nil, "", "", err
	}

	return file, "direct", url, nil
}

// GetUploadURL 生成用于上传文件的预签名 URL。
func (s *FileUsecase) GetUploadURL(ctx context.Context, schoolID, uploaderID, fileName, fileType string, size int64) (*storage.File, string, error) {
	file, _, url, err := s.InitUpload(ctx, schoolID, uploaderID, fileName, fileType, size)
	if err != nil {
		return nil, "", err
	}
	return file, url, nil
}

// GetDownloadInfo 返回文件记录，以及基于当前主 OSS 凭据的下载方式。
// - downloadMethod = "direct"：返回预签名 GET URL。
// - downloadMethod = "relay"：调用方应使用服务端中转下载接口。
func (s *FileUsecase) GetDownloadInfo(ctx context.Context, schoolID, fileID string) (*storage.File, string, string, error) {
	if strings.TrimSpace(fileID) == "" {
		return nil, "", "", fmt.Errorf("file id required")
	}

	file, err := s.files.GetByID(ctx, fileID, schoolID)
	if err != nil {
		return nil, "", "", err
	}

	cred, err := s.getPrimaryCredential(ctx, schoolID)
	if err != nil {
		return nil, "", "", err
	}
	if cred.UseRelayUpload {
		return file, "relay", "", nil
	}

	client, err := s.newClientWithEndpoint(cred, s.downloadEndpoint(cred))
	if err != nil {
		return nil, "", "", err
	}
	url, err := client.SignURL(file.Key, "GET", 3600)
	if err != nil {
		return nil, "", "", err
	}
	return file, "direct", url, nil
}

// OpenDownloadStream 打开 OSS 对象流，用于服务端中转下载。
func (s *FileUsecase) OpenDownloadStream(ctx context.Context, schoolID, fileID string) (*storage.File, io.ReadCloser, error) {
	if strings.TrimSpace(fileID) == "" {
		return nil, nil, fmt.Errorf("file id required")
	}
	file, err := s.files.GetByID(ctx, fileID, schoolID)
	if err != nil {
		return nil, nil, err
	}

	cred, err := s.getPrimaryCredential(ctx, schoolID)
	if err != nil {
		return nil, nil, err
	}
	client, err := s.newClientWithEndpoint(cred, s.uploadEndpoint(cred))
	if err != nil {
		return nil, nil, err
	}

	rc, err := client.GetObject(file.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get oss object: %w", err)
	}
	return file, rc, nil
}

// AttachToAssignment 将文件关联到作业。
func (s *FileUsecase) AttachToAssignment(ctx context.Context, assignmentID, fileID string) error {
	attachment := &coursework.AssignmentAttachment{
		ID:           uuid.New().String(),
		AssignmentID: assignmentID,
		FileID:       fileID,
		CreatedAt:    time.Now(),
	}
	return s.attachments.CreateAssignmentAttachment(ctx, attachment)
}

// AttachToSubmission 将文件关联到提交。
func (s *FileUsecase) AttachToSubmission(ctx context.Context, submissionID, fileID string) error {
	attachment := &coursework.SubmissionAttachment{
		ID:           uuid.New().String(),
		SubmissionID: submissionID,
		FileID:       fileID,
		CreatedAt:    time.Now(),
	}
	return s.attachments.CreateSubmissionAttachment(ctx, attachment)
}
