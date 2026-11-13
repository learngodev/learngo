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
	// 注意：真正使用 STS 需要单独的 STS 客户端和角色 ARN。
	// 由于配置中只使用了普通 OSS 密钥，在没有 RAM 角色的情况下
	// 可能无法生成 STS 令牌。
	// 为简化本 "learn-go" 项目，在未配置 STS 时上传可回退为预签名 URL，
	// 或直接返回错误。
	// 但接口要求提供该方法。
	// 目前本项目主要使用预签名 URL，
	// 因为这比配置 RAM 角色更简单。
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
