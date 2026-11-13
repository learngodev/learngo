package storage

import (
	"time"
)

// File 表示上传到 OSS 的文件。
type File struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	SchoolID   string    `gorm:"size:36;index" json:"school_id"`
	UploaderID string    `gorm:"size:36;index" json:"uploader_id"`
	Name       string    `gorm:"size:256" json:"name"`
	Key        string    `gorm:"size:256" json:"key"`  // OSS 对象 Key
	URL        string    `gorm:"size:512" json:"url"`  // 公开 URL 或预签名 URL
	Type       string    `gorm:"size:255" json:"type"` // MIME 类型
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
}
