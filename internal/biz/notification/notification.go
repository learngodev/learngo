package notification

import "time"

// NotificationType defines the type of notification.
type NotificationType string

const (
	NotificationTypeAssignment NotificationType = "assignment"
	NotificationTypeSystem     NotificationType = "system"
)

// Notification represents a user notification.
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
