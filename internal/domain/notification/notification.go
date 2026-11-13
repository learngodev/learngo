package notification

import (
	"time"
)

// NotificationType 定义通知类型。
type NotificationType string

const (
	NotificationTypeAssignment NotificationType = "assignment"
	NotificationTypeSystem     NotificationType = "system"
)

// Notification 表示一条用户通知。
type Notification struct {
	ID          string           `gorm:"primaryKey;size:36" json:"id"`
	UserID      string           `gorm:"size:36;index" json:"user_id"`
	Title       string           `gorm:"size:256" json:"title"`
	Content     string           `gorm:"type:text" json:"content"`
	Type        NotificationType `gorm:"size:32" json:"type"`
	ReferenceID string           `gorm:"size:36" json:"reference_id"`
	IsRead      bool             `gorm:"default:false" json:"is_read"`
	CreatedAt   time.Time        `json:"created_at"`
}
