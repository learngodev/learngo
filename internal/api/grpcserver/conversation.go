package grpcserver

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"learn-go/internal/api/grpcmapper"
	"learn-go/internal/api/grpcpb"
	"learn-go/internal/realtime"
	"learn-go/internal/service"
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

// Stream handles bidirectional conversation streams.
func (s *ConversationServer) Stream(stream grpcpb.ConversationService_StreamServer) error {
	ctx := stream.Context()
	accountID, ok := AccountIDFromContext(ctx)
	if !ok || accountID == "" {
		return status.Error(codes.Unauthenticated, "missing account context")
	}

	var (
		joined         bool
		conversationID string
		client         *realtime.StreamClient
		sendErrCh      chan error
	)

	for {
		if joined && sendErrCh != nil {
			select {
			case err := <-sendErrCh:
				return err
			default:
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if !joined {
			join := req.GetJoin()
			if join == nil {
				return status.Error(codes.InvalidArgument, "expected join payload as first message")
			}
			conversationID = strings.TrimSpace(join.GetConversationId())
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

			client = realtime.NewStreamClient(conversationID)
			s.hub.Register(client)
			defer s.hub.Unregister(client)

			snapshot := &grpcpb.ConversationStreamResponse{
				Payload: &grpcpb.ConversationStreamResponse_Snapshot{
					Snapshot: grpcmapper.Snapshot(*summary, messages),
				},
			}
			_ = client.Send(snapshot)

			sendErrCh = make(chan error, 1)
			go forwardStream(stream, client, sendErrCh)

			joined = true
			continue
		}

		switch payload := req.GetPayload().(type) {
		case *grpcpb.ConversationStreamRequest_Read:
			s.handleRead(ctx, client, accountID, conversationID, payload.Read)
		case *grpcpb.ConversationStreamRequest_Create:
			s.handleCreate(ctx, client, accountID, conversationID, payload.Create)
		default:
			_ = client.Send(errorResponse("unsupported payload"))
		}
	}
}

func (s *ConversationServer) handleRead(ctx context.Context, client *realtime.StreamClient, accountID, conversationID string, payload *grpcpb.ConversationRead) {
	if payload == nil || strings.TrimSpace(payload.GetMessageId()) == "" {
		_ = client.Send(errorResponse("message_id required"))
		return
	}

	ctxTimeout, cancel := withTimeout(ctx)
	defer cancel()

	err := s.conversations.MarkRead(ctxTimeout, accountID, conversationID, payload.GetMessageId())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			_ = client.Send(errorResponse("not allowed to mark read"))
		case errors.Is(err, service.ErrConversationNotFound):
			_ = client.Send(errorResponse("conversation or message not found"))
		default:
			_ = client.Send(errorResponse("unable to mark read"))
		}
		return
	}

	ack := &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_ReadAck{
			ReadAck: &grpcpb.ConversationReadAck{MessageId: payload.GetMessageId()},
		},
	}
	_ = client.Send(ack)

	s.hub.Broadcast(conversationID, &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_ReadEvent{
			ReadEvent: &grpcpb.ConversationReadEvent{
				MessageId: payload.GetMessageId(),
				ReaderId:  accountID,
			},
		},
	})
}

func (s *ConversationServer) handleCreate(ctx context.Context, client *realtime.StreamClient, accountID, conversationID string, payload *grpcpb.MessageCreate) {
	if payload == nil {
		_ = client.Send(errorResponse("message payload required"))
		return
	}

	if strings.TrimSpace(payload.GetConversationId()) != "" && strings.TrimSpace(payload.GetConversationId()) != conversationID {
		_ = client.Send(errorResponse("conversation mismatch"))
		return
	}

	kind := strings.TrimSpace(payload.GetKind())
	if kind == "" {
		_ = client.Send(errorResponse("kind is required"))
		return
	}

	if !isSupportedKind(kind) {
		_ = client.Send(errorResponse("unsupported message kind"))
		return
	}

	if kind == "text" {
		if strings.TrimSpace(payload.GetText()) == "" {
			_ = client.Send(errorResponse("text message requires content"))
			return
		}
	} else if strings.TrimSpace(payload.GetMediaUri()) == "" {
		_ = client.Send(errorResponse("media message requires media_uri"))
		return
	}

	ctxTimeout, cancel := withTimeout(ctx)
	defer cancel()

	msg, err := s.conversations.SendMessage(ctxTimeout, accountID, service.SendMessageInput{
		ConversationID: conversationID,
		Kind:           kind,
		Text:           payload.GetText(),
		MediaURI:       payload.GetMediaUri(),
		Metadata:       payload.GetMetadata(),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			_ = client.Send(errorResponse("not allowed to send message"))
		case errors.Is(err, service.ErrConversationNotFound):
			_ = client.Send(errorResponse("conversation not found"))
		default:
			_ = client.Send(errorResponse("unable to send message"))
		}
		return
	}

	messageProto := grpcmapper.Message(*msg)

	ack := &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_MessageAck{
			MessageAck: &grpcpb.MessageCreateAck{Message: messageProto},
		},
	}
	_ = client.Send(ack)

	s.hub.Broadcast(conversationID, &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_MessageEvent{
			MessageEvent: &grpcpb.MessageCreatedEvent{Message: messageProto},
		},
	})
}

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Second)
}

func forwardStream(stream grpcpb.ConversationService_StreamServer, client *realtime.StreamClient, errCh chan<- error) {
	for {
		select {
		case <-stream.Context().Done():
			client.Close()
			errCh <- stream.Context().Err()
			return
		case resp, ok := <-client.Messages():
			if !ok {
				errCh <- nil
				return
			}
			if err := stream.Send(resp); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func errorResponse(message string) *grpcpb.ConversationStreamResponse {
	return &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_Error{
			Error: &grpcpb.Error{Message: message},
		},
	}
}

func isSupportedKind(kind string) bool {
	switch kind {
	case "text", "image", "video", "audio", "file":
		return true
	default:
		return false
	}
}
