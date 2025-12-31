package grpcserver

import (
	"learn-go/internal/api/grpcpb"
	"learn-go/internal/realtime"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NotificationServer struct {
	grpcpb.UnimplementedNotificationServiceServer
	hub *realtime.NotificationHub
}

func NewNotificationServer(hub *realtime.NotificationHub) *NotificationServer {
	return &NotificationServer{hub: hub}
}

func (s *NotificationServer) Subscribe(_ *emptypb.Empty, stream grpcpb.NotificationService_SubscribeServer) error {
	ctx := stream.Context()
	userID, ok := AccountIDFromContext(ctx)
	if !ok || userID == "" {
		return status.Error(codes.Unauthenticated, "missing account context")
	}

	client := realtime.NewNotificationStream(userID)
	s.hub.Register(client)
	defer s.hub.Unregister(client)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-client.Messages():
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}
}
