package service

import (
	"context"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"learn-go/api/grpcpb"
	notificationbiz "learn-go/internal/biz/notification"
	"learn-go/internal/realtime"
	"time"
)

type NotificationService struct {
	notifications notificationbiz.NotificationRepository
	hub           *realtime.NotificationHub
}

func NewNotificationService(notifications notificationbiz.NotificationRepository, hub *realtime.NotificationHub) *NotificationService {
	return &NotificationService{notifications: notifications, hub: hub}
}

func (s *NotificationService) Create(ctx context.Context, userID, title, content string, notifType notificationbiz.NotificationType, refID string) error {
	notification := &notificationbiz.Notification{
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

	// Push to gRPC stream
	if s.hub != nil {
		s.hub.SendToUser(userID, &grpcpb.NotificationStreamResponse{
			Payload: &grpcpb.NotificationStreamResponse_Notification{
				Notification: &grpcpb.NotificationEvent{
					Id:          notification.ID,
					Title:       notification.Title,
					Content:     notification.Content,
					Type:        string(notification.Type),
					ReferenceId: notification.ReferenceID,
					IsRead:      notification.IsRead,
					CreatedAt:   timestamppb.New(notification.CreatedAt),
				},
			},
		})
	}

	return nil
}

func (s *NotificationService) ListByUser(ctx context.Context, userID string, limit, offset int) ([]notificationbiz.Notification, int64, error) {
	return s.notifications.ListByUser(ctx, userID, limit, offset)
}

func (s *NotificationService) MarkAsRead(ctx context.Context, id, userID string) error {
	return s.notifications.MarkAsRead(ctx, id, userID)
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.notifications.MarkAllAsRead(ctx, userID)
}

func (s *NotificationService) CountUnread(ctx context.Context, userID string) (int64, error) {
	return s.notifications.CountUnread(ctx, userID)
}
