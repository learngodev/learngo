package oss

import (
	"context"
	"io"
)

// Client defines interface for interacting with Object Storage Service.
type Client interface {
	// GenerateUploadCredentials returns STS credentials for client-side upload.
	GenerateUploadCredentials(ctx context.Context, sessionName string, durationSeconds int64) (*UploadCredentials, error)

	// SignURL generates a presigned URL for uploading or downloading.
	// method: "GET" or "PUT"
	SignURL(objectKey string, method string, expiredInSec int64) (string, error)

	// PutObject uploads data directly from server.
	PutObject(objectKey string, reader io.Reader) error

	// DeleteObject deletes an object.
	DeleteObject(objectKey string) error
}

// UploadCredentials represent temporary upload data for clients.
type UploadCredentials struct {
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	SecurityToken   string `json:"security_token"`
	Expiration      string `json:"expiration"`
}

// StaticClient returns preconfigured credentials (placeholder for integration).
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

func (c *StaticClient) DeleteObject(objectKey string) error {
	return nil
}
