package grpcserver

import (
	"learn-go/internal/domain/notification"
	"learn-go/internal/interface/grpcpb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NotificationServer struct {
	grpcpb.UnimplementedNotificationServiceServer
	subscriber notification.Subscriber
}

func NewNotificationServer(subscriber notification.Subscriber) *NotificationServer {
	return &NotificationServer{subscriber: subscriber}
}

func (s *NotificationServer) Subscribe(_ *emptypb.Empty, stream grpcpb.NotificationService_SubscribeServer) error {
	ctx := stream.Context()
	userID, ok := AccountIDFromContext(ctx)
	if !ok || userID == "" {
		return status.Error(codes.Unauthenticated, "missing account context")
	}

	notifications, cancel := s.subscriber.Subscribe(userID)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case n, ok := <-notifications:
			if !ok {
				return nil
			}
			if err := stream.Send(toNotificationStreamResponse(n)); err != nil {
				return err
			}
		}
	}
}

func toNotificationStreamResponse(n *notification.Notification) *grpcpb.NotificationStreamResponse {
	return &grpcpb.NotificationStreamResponse{
		Payload: &grpcpb.NotificationStreamResponse_Notification{
			Notification: &grpcpb.NotificationEvent{
				Id:          n.ID,
				Title:       n.Title,
				Content:     n.Content,
				Type:        string(n.Type),
				ReferenceId: n.ReferenceID,
				IsRead:      n.IsRead,
				CreatedAt:   timestamppb.New(n.CreatedAt),
			},
		},
	}
}
