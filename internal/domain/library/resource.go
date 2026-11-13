package library

import (
	"time"

	"gorm.io/gorm"
)

// Resource 表示教师上传的教学资源。
type Resource struct {
	ID            string         `gorm:"primaryKey;size:36" json:"id"`
	SchoolID      string         `gorm:"size:36;index:idx_school_teacher" json:"school_id"`
	TeacherID     string         `gorm:"size:36;index:idx_school_teacher" json:"teacher_id"`
	DepartmentID  *string        `gorm:"size:36;index" json:"department_id"`
	Title         string         `gorm:"size:256" json:"title"`
	Description   string         `gorm:"type:text" json:"description"`
	GradeLevel    string         `gorm:"size:32" json:"grade_level"`
	Tags          string         `gorm:"type:text" json:"tags"` // JSON 数组
	ViewCount     int            `gorm:"default:0" json:"view_count"`
	DownloadCount int            `gorm:"default:0" json:"download_count"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// 虚拟字段（只读）
	TeacherName    string `gorm:"->;<-:false" json:"teacher_name"`
	DepartmentName string `gorm:"->;<-:false" json:"department_name"`
	FileCount      int    `gorm:"->;<-:false" json:"file_count"`
	FavoriteCount  int    `gorm:"->;<-:false" json:"favorite_count"`
	IsFavorited    bool   `gorm:"->;<-:false" json:"is_favorited"`
}

// ResourceFile 关联资源与文件。
type ResourceFile struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	ResourceID string    `gorm:"size:36;index" json:"resource_id"`
	FileID     string    `gorm:"size:36;index" json:"file_id"`
	OrderIndex int       `gorm:"default:0" json:"order_index"`
	CreatedAt  time.Time `json:"created_at"`
}

// ResourceFavorite 记录用户收藏。
type ResourceFavorite struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	ResourceID string    `gorm:"size:36;uniqueIndex:idx_resource_account" json:"resource_id"`
	AccountID  string    `gorm:"size:36;uniqueIndex:idx_resource_account;index" json:"account_id"`
	CreatedAt  time.Time `json:"created_at"`
}
