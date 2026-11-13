package objectstorage

import (
	"learn-go/internal/domain/storage"
	"learn-go/pkg/oss"
)

// AliyunFactory 基于阿里云 OSS 实现 storage.ObjectStorageFactory。
type AliyunFactory struct{}

// NewAliyunFactory 创建 AliyunFactory。
func NewAliyunFactory() AliyunFactory { return AliyunFactory{} }

func (AliyunFactory) NewClient(endpoint string, cred *storage.OssCredential) (storage.ObjectStorage, error) {
	return oss.NewAliyunClient(endpoint, cred.AccessKeyID, cred.AccessKeySecret, cred.Bucket)
}

var _ storage.ObjectStorageFactory = AliyunFactory{}
