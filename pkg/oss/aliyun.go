package oss

import (
	"context"
	"fmt"
	"io"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type AliyunClient struct {
	client          *oss.Client
	bucket          *oss.Bucket
	endpoint        string
	bucketName      string
	accessKeyID     string
	accessKeySecret string
}

func NewAliyunClient(endpoint, accessKeyID, accessKeySecret, bucketName string) (*AliyunClient, error) {
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create oss client: %w", err)
	}

	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &AliyunClient{
		client:          client,
		bucket:          bucket,
		endpoint:        endpoint,
		bucketName:      bucketName,
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
	}, nil
}

func (c *AliyunClient) GenerateUploadCredentials(ctx context.Context, sessionName string, durationSeconds int64) (*UploadCredentials, error) {
	// Note: For real STS, we need a separate STS client and role ARN.
	// Since we are using simple OSS keys in config, we might not be able to generate STS tokens
	// without an RAM role.
	// For simplicity in this "learn-go" project, we might fallback to Presigned URL for uploads
	// if STS is not configured, or just return error.
	// However, the interface asks for it.
	// Let's assume for now we use Presigned URLs primarily for this project structure
	// as it's simpler than setting up RAM roles.
	return nil, fmt.Errorf("STS not implemented, use SignURL for uploads")
}

func (c *AliyunClient) SignURL(objectKey string, method string, expiredInSec int64) (string, error) {
	var options []oss.Option
	httpMethod := oss.HTTPGet
	if method == "PUT" {
		httpMethod = oss.HTTPPut
		options = append(options, oss.ContentType("application/octet-stream"))
	}

	signedURL, err := c.bucket.SignURL(objectKey, httpMethod, expiredInSec, options...)
	if err != nil {
		return "", fmt.Errorf("failed to sign url: %w", err)
	}
	return signedURL, nil
}

func (c *AliyunClient) PutObject(objectKey string, reader io.Reader) error {
	return c.bucket.PutObject(objectKey, reader)
}

func (c *AliyunClient) GetObject(objectKey string) (io.ReadCloser, error) {
	return c.bucket.GetObject(objectKey)
}

func (c *AliyunClient) DeleteObject(objectKey string) error {
	return c.bucket.DeleteObject(objectKey)
}
