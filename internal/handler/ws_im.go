package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"

	"learn-go/internal/interface/grpcpb"
	"learn-go/internal/interface/imstream"
	"learn-go/pkg/middleware"
	"learn-go/pkg/response"
)

// IMWebSocket 将连接升级为 WebSocket，并在其上运行会话双向流。
//
// 协议：
// - 客户端必须发送二进制 protobuf 格式的 ConversationStreamRequest。
// - 第一条消息必须是 JoinConversation 负载。
// - 服务端以二进制帧推送 protobuf 格式的 ConversationStreamResponse。
func (h *Handler) IMWebSocket(c *gin.Context) {
	token := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		// 浏览器的 WebSocket 无法设置自定义请求头，因此允许通过查询参数传递 token。
		// 注意：代理可能会记录该参数；生产环境建议使用安全 cookie 或短期有效的 WS 票据。
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

	accountID, role, _, err := middleware.ValidateJWT(token, h.jwtSecret)
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
				// 客户端关闭时 context 被取消属于正常情况。
				if errors.Is(err, context.Canceled) {
					return
				}
				return
			default:
			}
		}

		typ, data, err := conn.Read(ctx)
		if err != nil {
			// 正常关闭或 context 被取消。
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
				// 加入失败时直接关闭连接，以保持协议简单。
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
