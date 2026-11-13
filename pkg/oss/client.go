package oss

import (
	"context"
	"io"
	"strings"
)

// Client 定义与对象存储服务交互的接口。
type Client interface {
	// GenerateUploadCredentials 返回用于客户端直传的 STS 凭据。
	GenerateUploadCredentials(ctx context.Context, sessionName string, durationSeconds int64) (*UploadCredentials, error)

	// SignURL 生成用于上传或下载的预签名 URL。
	// method："GET" 或 "PUT"
	SignURL(objectKey string, method string, expiredInSec int64) (string, error)

	// PutObject 由服务端直接上传数据。
	PutObject(objectKey string, reader io.Reader) error

	// GetObject 下载对象内容，用于服务端中转。
	// 调用方必须关闭返回的 ReadCloser。
	GetObject(objectKey string) (io.ReadCloser, error)

	// DeleteObject 删除对象。
	DeleteObject(objectKey string) error
}

// UploadCredentials 表示提供给客户端的临时上传凭据。
type UploadCredentials struct {
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	SecurityToken   string `json:"security_token"`
	Expiration      string `json:"expiration"`
}

// StaticClient 返回预置的凭据（集成前的占位实现）。
type StaticClient struct {
	Endpoint string
	Bucket   string
}

func (c *StaticClient) GenerateUploadCredentials(ctx context.Context, sessionName string, durationSeconds int64) (*UploadCredentials, error) {
	return &UploadCredentials{
		Endpoint: c.Endpoint,
		Bucket:   c.Bucket,
	}, nil
}

func (c *StaticClient) SignURL(objectKey string, method string, expiredInSec int64) (string, error) {
	return c.Endpoint + "/" + c.Bucket + "/" + objectKey, nil
}

func (c *StaticClient) PutObject(objectKey string, reader io.Reader) error {
	return nil
}

func (c *StaticClient) GetObject(objectKey string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (c *StaticClient) DeleteObject(objectKey string) error {
	return nil
}
