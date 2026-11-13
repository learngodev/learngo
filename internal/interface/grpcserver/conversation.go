package grpcserver

import (
	"errors"
	"io"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"learn-go/internal/interface/grpcmapper"
	"learn-go/internal/interface/grpcpb"
	"learn-go/internal/interface/imstream"
	"learn-go/internal/usecase"
)

const conversationHistoryLimit = 50

// ConversationServer 实现 gRPC 会话流服务。
type ConversationServer struct {
	grpcpb.UnimplementedConversationServiceServer

	conversations *usecase.ConversationUsecase
	hub           *imstream.Hub
}

// NewConversationServer 构造 ConversationServer。
func NewConversationServer(conversations *usecase.ConversationUsecase, hub *imstream.Hub) *ConversationServer {
	return &ConversationServer{conversations: conversations, hub: hub}
}

// Subscribe 处理服务端流式的会话订阅（对 Web 友好）。
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
		case errors.Is(err, usecase.ErrConversationForbidden):
			return status.Error(codes.PermissionDenied, "not a member of this conversation")
		case errors.Is(err, usecase.ErrConversationNotFound):
			return status.Error(codes.NotFound, "conversation not found")
		default:
			return status.Errorf(codes.Internal, "load conversation summary: %v", err)
		}
	}

	messages, err := s.conversations.ListMessages(ctx, accountID, conversationID, conversationHistoryLimit, "")
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrConversationForbidden):
			return status.Error(codes.PermissionDenied, "not allowed to view messages")
		case errors.Is(err, usecase.ErrConversationNotFound):
			return status.Error(codes.NotFound, "conversation not found")
		default:
			return status.Errorf(codes.Internal, "load conversation messages: %v", err)
		}
	}

	client := imstream.NewStreamClient(conversationID)
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

// SubscribeInbox 推送当前认证用户所有会话中的事件。
// 这样客户端（尤其是通过 gRPC-Web 接入的 Web 端）只需一个流式 RPC，
// 即可保持会话列表和未读角标的实时更新。
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
	clients := make([]*imstream.StreamClient, 0, len(summaries))

	for _, summary := range summaries {
		conversationID := summary.Conversation.ID
		if strings.TrimSpace(conversationID) == "" {
			continue
		}

		client := imstream.NewStreamClient(conversationID)
		s.hub.Register(client)
		clients = append(clients, client)

		go func(c *imstream.StreamClient) {
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

// Stream 处理会话双向流。
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
