package storage

import "io"

// ObjectStorage 定义与对象存储服务交互的能力。
type ObjectStorage interface {
	// SignURL 生成预签名 URL，method 为 "GET" 或 "PUT"。
	SignURL(objectKey string, method string, expiredInSec int64) (string, error)
	PutObject(objectKey string, reader io.Reader) error
	// GetObject 返回对象内容，调用方必须关闭返回的 ReadCloser。
	GetObject(objectKey string) (io.ReadCloser, error)
	DeleteObject(objectKey string) error
}

// ObjectStorageFactory 根据凭据与 endpoint 创建对象存储客户端。
type ObjectStorageFactory interface {
	NewClient(endpoint string, cred *OssCredential) (ObjectStorage, error)
}
