package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"

	"learn-go/internal/api/grpcpb"
	"learn-go/internal/api/imstream"
	"learn-go/pkg/middleware"
	"learn-go/pkg/response"
)

// IMWebSocket upgrades the connection and runs a conversation bidirectional stream over WebSocket.
//
// Protocol:
// - Client MUST send binary protobuf ConversationStreamRequest.
// - First message MUST be JoinConversation payload.
// - Server pushes protobuf ConversationStreamResponse as binary frames.
func (h *Handler) IMWebSocket(c *gin.Context) {
	token := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		// Browser WebSocket cannot set custom headers; allow token in query.
		// NOTE: this may be logged by proxies; for production prefer secure cookie or a short-lived WS ticket.
		token = strings.TrimSpace(c.Query("token"))
		if token == "" {
			token = strings.TrimSpace(c.Query("access_token"))
		}
	}
	if token == "" {
		response.Error(c, http.StatusUnauthorized, "missing authorization", nil)
		c.Abort()
		return
	}

	accountID, role, err := middleware.ValidateJWT(token, h.jwtSecret)
	if err != nil || accountID == "" {
		response.Error(c, http.StatusUnauthorized, "invalid token", nil)
		c.Abort()
		return
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "student" && role != "teacher" && role != "admin" {
		response.Error(c, http.StatusForbidden, "insufficient role", nil)
		c.Abort()
		return
	}

	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx := c.Request.Context()

	session := imstream.NewSession(h.conversations, h.streamHub, accountID)
	defer session.Close()

	var writeMu sync.Mutex
	writeResp := func(resp *grpcpb.ConversationStreamResponse) error {
		if resp == nil {
			return nil
		}
		data, err := proto.Marshal(resp)
		if err != nil {
			return err
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.Write(ctx, websocket.MessageBinary, data)
	}

	sendClose := func(status websocket.StatusCode, reason string) {
		_ = conn.Close(status, reason)
	}

	forwardErrCh := make(chan error, 1)
	startedForward := false

	for {
		if startedForward {
			select {
			case err := <-forwardErrCh:
				if err == nil {
					return
				}
				// Context cancellation is normal on client close.
				if errors.Is(err, context.Canceled) {
					return
				}
				return
			default:
			}
		}

		typ, data, err := conn.Read(ctx)
		if err != nil {
			// Normal close or context cancel.
			return
		}
		if typ != websocket.MessageBinary {
			_ = writeResp(imstream.ErrorResponse("binary protobuf required"))
			continue
		}

		var req grpcpb.ConversationStreamRequest
		if err := proto.Unmarshal(data, &req); err != nil {
			_ = writeResp(imstream.ErrorResponse("invalid protobuf"))
			continue
		}

		wasJoined := session.Joined()
		if err := session.Handle(ctx, &req); err != nil {
			var se *imstream.StreamError
			if errors.As(err, &se) {
				_ = writeResp(imstream.ErrorResponse(se.Message))
				// If we cannot join, close to keep the protocol simple.
				if !session.Joined() {
					sendClose(websocket.StatusPolicyViolation, se.Message)
					return
				}
				continue
			}
			_ = writeResp(imstream.ErrorResponse("stream error"))
			continue
		}

		if !wasJoined && session.Joined() && !startedForward {
			startedForward = true
			go func() {
				forwardErrCh <- imstream.Forward(ctx, session.Client(), func(resp *grpcpb.ConversationStreamResponse) error {
					return writeResp(resp)
				})
			}()
		}
	}
}
