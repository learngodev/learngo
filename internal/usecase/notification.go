package usecase

import (
	"context"
	"time"

	"learn-go/internal/domain/notification"

	"github.com/google/uuid"
)

type NotificationUsecase struct {
	notifications notification.NotificationRepository
	publisher     notification.Publisher
}

func NewNotificationUsecase(notifications notification.NotificationRepository, publisher notification.Publisher) *NotificationUsecase {
	return &NotificationUsecase{notifications: notifications, publisher: publisher}
}

func (s *NotificationUsecase) Create(ctx context.Context, userID, title, content string, notifType notification.NotificationType, refID string) error {
	notification := &notification.Notification{
		ID:          uuid.NewString(),
		UserID:      userID,
		Title:       title,
		Content:     content,
		Type:        notifType,
		ReferenceID: refID,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}
	if err := s.notifications.Create(ctx, notification); err != nil {
		return err
	}

	// 实时推送给在线用户
	if s.publisher != nil {
		s.publisher.Publish(userID, notification)
	}

	return nil
}

func (s *NotificationUsecase) ListByUser(ctx context.Context, userID string, limit, offset int) ([]notification.Notification, int64, error) {
	return s.notifications.ListByUser(ctx, userID, limit, offset)
}

func (s *NotificationUsecase) MarkAsRead(ctx context.Context, id, userID string) error {
	return s.notifications.MarkAsRead(ctx, id, userID)
}

func (s *NotificationUsecase) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.notifications.MarkAllAsRead(ctx, userID)
}

func (s *NotificationUsecase) CountUnread(ctx context.Context, userID string) (int64, error) {
	return s.notifications.CountUnread(ctx, userID)
}
