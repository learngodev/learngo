package grpcserver

import (
	"errors"
	"io"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"learn-go/api/grpcpb"
	"learn-go/internal/handler/grpcmapper"
	"learn-go/internal/handler/imstream"
	"learn-go/internal/realtime"
	"learn-go/internal/usecase"
)

const conversationHistoryLimit = 50

// ConversationServer implements the gRPC conversation streaming service.
type ConversationServer struct {
	grpcpb.UnimplementedConversationServiceServer

	conversations *service.ConversationService
	hub           *realtime.Hub
}

// NewConversationServer constructs a ConversationServer.
func NewConversationServer(conversations *service.ConversationService, hub *realtime.Hub) *ConversationServer {
	return &ConversationServer{conversations: conversations, hub: hub}
}

// Subscribe handles server-streaming conversation subscriptions (web-friendly).
func (s *ConversationServer) Subscribe(req *grpcpb.JoinConversation, stream grpcpb.ConversationService_SubscribeServer) error {
	ctx := stream.Context()
	accountID, ok := AccountIDFromContext(ctx)
	if !ok || accountID == "" {
		return status.Error(codes.Unauthenticated, "missing account context")
	}

	conversationID := strings.TrimSpace(req.GetConversationId())
	if conversationID == "" {
		return status.Error(codes.InvalidArgument, "conversation_id is required")
	}

	summary, err := s.conversations.GetConversationSummary(ctx, accountID, conversationID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			return status.Error(codes.PermissionDenied, "not a member of this conversation")
		case errors.Is(err, service.ErrConversationNotFound):
			return status.Error(codes.NotFound, "conversation not found")
		default:
			return status.Errorf(codes.Internal, "load conversation summary: %v", err)
		}
	}

	messages, err := s.conversations.ListMessages(ctx, accountID, conversationID, conversationHistoryLimit, "")
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			return status.Error(codes.PermissionDenied, "not allowed to view messages")
		case errors.Is(err, service.ErrConversationNotFound):
			return status.Error(codes.NotFound, "conversation not found")
		default:
			return status.Errorf(codes.Internal, "load conversation messages: %v", err)
		}
	}

	client := realtime.NewStreamClient(conversationID)
	s.hub.Register(client)
	defer s.hub.Unregister(client)

	snapshot := &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_Snapshot{
			Snapshot: grpcmapper.Snapshot(*summary, messages),
		},
	}
	if err := stream.Send(snapshot); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			client.Close()
			return ctx.Err()
		case resp, ok := <-client.Messages():
			if !ok {
				return nil
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

// SubscribeInbox streams conversation events across all conversations for the authenticated user.
// This allows clients (especially web via gRPC-Web) to keep the conversation list and unread
// badges updated with a single streaming RPC.
func (s *ConversationServer) SubscribeInbox(_ *grpcpb.JoinConversation, stream grpcpb.ConversationService_SubscribeInboxServer) error {
	ctx := stream.Context()
	accountID, ok := AccountIDFromContext(ctx)
	if !ok || accountID == "" {
		return status.Error(codes.Unauthenticated, "missing account context")
	}

	summaries, err := s.conversations.ListConversations(ctx, accountID)
	if err != nil {
		return status.Errorf(codes.Internal, "list conversations: %v", err)
	}

	merged := make(chan *grpcpb.ConversationStreamResponse, 64)
	clients := make([]*realtime.StreamClient, 0, len(summaries))

	for _, summary := range summaries {
		conversationID := summary.Conversation.ID
		if strings.TrimSpace(conversationID) == "" {
			continue
		}

		client := realtime.NewStreamClient(conversationID)
		s.hub.Register(client)
		clients = append(clients, client)

		go func(c *realtime.StreamClient) {
			for {
				select {
				case <-ctx.Done():
					return
				case resp, ok := <-c.Messages():
					if !ok {
						return
					}
					select {
					case merged <- resp:
					case <-ctx.Done():
						return
					}
				}
			}
		}(client)
	}

	defer func() {
		for _, c := range clients {
			s.hub.Unregister(c)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case resp := <-merged:
			if resp == nil {
				continue
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

// Stream handles bidirectional conversation streams.
func (s *ConversationServer) Stream(stream grpcpb.ConversationService_StreamServer) error {
	ctx := stream.Context()
	accountID, ok := AccountIDFromContext(ctx)
	if !ok || accountID == "" {
		return status.Error(codes.Unauthenticated, "missing account context")
	}

	session := imstream.NewSession(s.conversations, s.hub, accountID)
	defer session.Close()

	sendErrCh := make(chan error, 1)
	startedForward := false

	for {
		if startedForward {
			select {
			case err := <-sendErrCh:
				return err
			default:
			}
		}

		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		wasJoined := session.Joined()
		if err := session.Handle(ctx, req); err != nil {
			var se *imstream.StreamError
			if errors.As(err, &se) {
				switch se.Kind {
				case imstream.ErrorKindInvalidArgument:
					return status.Error(codes.InvalidArgument, se.Message)
				case imstream.ErrorKindPermissionDenied:
					return status.Error(codes.PermissionDenied, se.Message)
				case imstream.ErrorKindNotFound:
					return status.Error(codes.NotFound, se.Message)
				default:
					return status.Errorf(codes.Internal, "%s", se.Message)
				}
			}
			return status.Error(codes.Internal, "stream error")
		}

		if !wasJoined && session.Joined() && !startedForward {
			startedForward = true
			go func() {
				sendErrCh <- imstream.Forward(ctx, session.Client(), func(resp *grpcpb.ConversationStreamResponse) error {
					return stream.Send(resp)
				})
			}()
		}
	}
}
