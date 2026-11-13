package imstream

import (
	"sync"

	"learn-go/internal/interface/grpcpb"
)

// StreamClient 表示某个会话上的一个活跃 gRPC 流订阅。
type StreamClient struct {
	conversationID string
	send           chan *grpcpb.ConversationStreamResponse
	once           sync.Once
}

// NewStreamClient 为指定会话构造一个 StreamClient。
func NewStreamClient(conversationID string) *StreamClient {
	return &StreamClient{
		conversationID: conversationID,
		send:           make(chan *grpcpb.ConversationStreamResponse, 32),
	}
}

// ConversationID 返回该客户端订阅的会话 ID。
func (c *StreamClient) ConversationID() string {
	return c.conversationID
}

// Messages 返回该客户端的出站消息通道。
func (c *StreamClient) Messages() <-chan *grpcpb.ConversationStreamResponse {
	return c.send
}

// Send 将响应放入队列，等待投递给客户端。
func (c *StreamClient) Send(resp *grpcpb.ConversationStreamResponse) bool {
	select {
	case c.send <- resp:
		return true
	default:
		return false
	}
}

// Close 关闭客户端通道。
func (c *StreamClient) Close() {
	c.once.Do(func() {
		close(c.send)
	})
}

// Hub 管理所有活跃的会话流客户端。
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*StreamClient]struct{}
}

// NewHub 构造一个空的 Hub 实例。
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*StreamClient]struct{}),
	}
}

// Register 将流客户端注册到 Hub。
func (h *Hub) Register(client *StreamClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	convClients := h.clients[client.conversationID]
	if convClients == nil {
		convClients = make(map[*StreamClient]struct{})
		h.clients[client.conversationID] = convClients
	}
	convClients[client] = struct{}{}
}

// Unregister 从 Hub 中移除客户端并关闭其通道。
func (h *Hub) Unregister(client *StreamClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	convClients, ok := h.clients[client.conversationID]
	if !ok {
		client.Close()
		return
	}
	if _, exists := convClients[client]; exists {
		delete(convClients, client)
	}
	if len(convClients) == 0 {
		delete(h.clients, client.conversationID)
	}
	client.Close()
}

// Broadcast 将响应投递给订阅该会话的所有客户端。
func (h *Hub) Broadcast(conversationID string, resp *grpcpb.ConversationStreamResponse) {
	h.mu.RLock()
	clients := h.clients[conversationID]
	for client := range clients {
		if ok := client.Send(resp); !ok {
			go h.Unregister(client)
		}
	}
	h.mu.RUnlock()
}
