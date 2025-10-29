package realtime

import (
	"sync"

	"learn-go/internal/api/grpcpb"
)

// StreamClient represents an active gRPC stream subscription for a conversation.
type StreamClient struct {
	conversationID string
	send           chan *grpcpb.ConversationStreamResponse
	once           sync.Once
}

// NewStreamClient constructs a StreamClient for the given conversation.
func NewStreamClient(conversationID string) *StreamClient {
	return &StreamClient{
		conversationID: conversationID,
		send:           make(chan *grpcpb.ConversationStreamResponse, 32),
	}
}

// ConversationID returns the conversation identifier for the client.
func (c *StreamClient) ConversationID() string {
	return c.conversationID
}

// Messages exposes the outbound message channel for the client.
func (c *StreamClient) Messages() <-chan *grpcpb.ConversationStreamResponse {
	return c.send
}

// Send enqueues a response for delivery to the client.
func (c *StreamClient) Send(resp *grpcpb.ConversationStreamResponse) bool {
	select {
	case c.send <- resp:
		return true
	default:
		return false
	}
}

// Close terminates the client channel.
func (c *StreamClient) Close() {
	c.once.Do(func() {
		close(c.send)
	})
}

// Hub manages active conversation stream clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*StreamClient]struct{}
}

// NewHub constructs an empty Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*StreamClient]struct{}),
	}
}

// Register adds a stream client to the hub.
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

// Unregister removes a client from the hub and closes its channel.
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

// Broadcast delivers the response to all clients subscribed to the conversation.
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
