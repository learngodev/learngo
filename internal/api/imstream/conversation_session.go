package imstream

import (
	"context"
	"errors"
	"strings"
	"time"

	"learn-go/internal/api/grpcmapper"
	"learn-go/internal/api/grpcpb"
	"learn-go/internal/realtime"
	"learn-go/internal/service"
)

const conversationHistoryLimit = 50

type ErrorKind int

const (
	ErrorKindInvalidArgument ErrorKind = iota + 1
	ErrorKindPermissionDenied
	ErrorKindNotFound
	ErrorKindInternal
)

type StreamError struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *StreamError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Message
	}
	return e.Message + ": " + e.Err.Error()
}

func (e *StreamError) Unwrap() error { return e.Err }

// Session owns a single conversation stream lifecycle (first message must be join).
// It is transport-agnostic: gRPC bidi stream and WebSocket can both drive it.
type Session struct {
	conversations  *service.ConversationService
	hub            *realtime.Hub
	accountID      string
	joined         bool
	conversationID string
	client         *realtime.StreamClient
}

func NewSession(conversations *service.ConversationService, hub *realtime.Hub, accountID string) *Session {
	return &Session{conversations: conversations, hub: hub, accountID: accountID}
}

func (s *Session) Joined() bool { return s.joined }

func (s *Session) ConversationID() string { return s.conversationID }

func (s *Session) Client() *realtime.StreamClient { return s.client }

func (s *Session) Close() {
	if s.client != nil {
		s.hub.Unregister(s.client)
		s.client = nil
	}
}

func (s *Session) Handle(ctx context.Context, req *grpcpb.ConversationStreamRequest) error {
	if req == nil {
		return &StreamError{Kind: ErrorKindInvalidArgument, Message: "request required"}
	}

	if !s.joined {
		join := req.GetJoin()
		if join == nil {
			return &StreamError{Kind: ErrorKindInvalidArgument, Message: "expected join payload as first message"}
		}
		return s.join(ctx, join)
	}

	switch payload := req.GetPayload().(type) {
	case *grpcpb.ConversationStreamRequest_Read:
		s.handleRead(ctx, payload.Read)
	case *grpcpb.ConversationStreamRequest_Create:
		s.handleCreate(ctx, payload.Create)
	default:
		_ = s.client.Send(ErrorResponse("unsupported payload"))
	}

	return nil
}

func (s *Session) join(ctx context.Context, join *grpcpb.JoinConversation) error {
	conversationID := strings.TrimSpace(join.GetConversationId())
	if conversationID == "" {
		return &StreamError{Kind: ErrorKindInvalidArgument, Message: "conversation_id is required"}
	}

	summary, err := s.conversations.GetConversationSummary(ctx, s.accountID, conversationID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			return &StreamError{Kind: ErrorKindPermissionDenied, Message: "not a member of this conversation", Err: err}
		case errors.Is(err, service.ErrConversationNotFound):
			return &StreamError{Kind: ErrorKindNotFound, Message: "conversation not found", Err: err}
		default:
			return &StreamError{Kind: ErrorKindInternal, Message: "load conversation summary", Err: err}
		}
	}

	messages, err := s.conversations.ListMessages(ctx, s.accountID, conversationID, conversationHistoryLimit, "")
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			return &StreamError{Kind: ErrorKindPermissionDenied, Message: "not allowed to view messages", Err: err}
		case errors.Is(err, service.ErrConversationNotFound):
			return &StreamError{Kind: ErrorKindNotFound, Message: "conversation not found", Err: err}
		default:
			return &StreamError{Kind: ErrorKindInternal, Message: "load conversation messages", Err: err}
		}
	}

	s.conversationID = conversationID
	s.client = realtime.NewStreamClient(conversationID)
	s.hub.Register(s.client)
	s.joined = true

	snapshot := &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_Snapshot{
			Snapshot: grpcmapper.Snapshot(*summary, messages),
		},
	}
	_ = s.client.Send(snapshot)

	return nil
}

func (s *Session) handleRead(ctx context.Context, payload *grpcpb.ConversationRead) {
	if payload == nil || strings.TrimSpace(payload.GetMessageId()) == "" {
		_ = s.client.Send(ErrorResponse("message_id required"))
		return
	}

	ctxTimeout, cancel := withTimeout(ctx)
	defer cancel()

	err := s.conversations.MarkRead(ctxTimeout, s.accountID, s.conversationID, payload.GetMessageId())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			_ = s.client.Send(ErrorResponse("not allowed to mark read"))
		case errors.Is(err, service.ErrConversationNotFound):
			_ = s.client.Send(ErrorResponse("conversation or message not found"))
		default:
			_ = s.client.Send(ErrorResponse("unable to mark read"))
		}
		return
	}

	ack := &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_ReadAck{
			ReadAck: &grpcpb.ConversationReadAck{MessageId: payload.GetMessageId()},
		},
	}
	_ = s.client.Send(ack)

	s.hub.Broadcast(s.conversationID, &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_ReadEvent{
			ReadEvent: &grpcpb.ConversationReadEvent{
				MessageId: payload.GetMessageId(),
				ReaderId:  s.accountID,
			},
		},
	})
}

func (s *Session) handleCreate(ctx context.Context, payload *grpcpb.MessageCreate) {
	if payload == nil {
		_ = s.client.Send(ErrorResponse("message payload required"))
		return
	}

	if strings.TrimSpace(payload.GetConversationId()) != "" && strings.TrimSpace(payload.GetConversationId()) != s.conversationID {
		_ = s.client.Send(ErrorResponse("conversation mismatch"))
		return
	}

	kind := strings.TrimSpace(payload.GetKind())
	if kind == "" {
		_ = s.client.Send(ErrorResponse("kind is required"))
		return
	}

	if !isSupportedKind(kind) {
		_ = s.client.Send(ErrorResponse("unsupported message kind"))
		return
	}

	if kind == "text" {
		if strings.TrimSpace(payload.GetText()) == "" {
			_ = s.client.Send(ErrorResponse("text message requires content"))
			return
		}
	} else if strings.TrimSpace(payload.GetMediaUri()) == "" {
		_ = s.client.Send(ErrorResponse("media message requires media_uri"))
		return
	}

	ctxTimeout, cancel := withTimeout(ctx)
	defer cancel()

	msg, err := s.conversations.SendMessage(ctxTimeout, s.accountID, service.SendMessageInput{
		ConversationID: s.conversationID,
		Kind:           kind,
		Text:           payload.GetText(),
		MediaURI:       payload.GetMediaUri(),
		Metadata:       payload.GetMetadata(),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConversationForbidden):
			_ = s.client.Send(ErrorResponse("not allowed to send message"))
		case errors.Is(err, service.ErrConversationNotFound):
			_ = s.client.Send(ErrorResponse("conversation not found"))
		default:
			_ = s.client.Send(ErrorResponse("unable to send message"))
		}
		return
	}

	messageProto := grpcmapper.Message(*msg)

	ack := &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_MessageAck{
			MessageAck: &grpcpb.MessageCreateAck{Message: messageProto},
		},
	}
	_ = s.client.Send(ack)

	s.hub.Broadcast(s.conversationID, &grpcpb.ConversationStreamResponse{
		Payload: &grpcpb.ConversationStreamResponse_MessageEvent{
			MessageEvent: &grpcpb.MessageCreatedEvent{Message: messageProto},
		},
	})
}

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Second)
}

func ErrorResponse(message string) *grpcpb.ConversationStreamResponse {
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
