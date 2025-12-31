package realtime

import (
	"learn-go/internal/api/grpcpb"
	"sync"
)

type NotificationStream struct {
	userID string
	send   chan *grpcpb.NotificationStreamResponse
	once   sync.Once
}

func NewNotificationStream(userID string) *NotificationStream {
	return &NotificationStream{
		userID: userID,
		send:   make(chan *grpcpb.NotificationStreamResponse, 32),
	}
}

func (s *NotificationStream) UserID() string                                      { return s.userID }
func (s *NotificationStream) Messages() <-chan *grpcpb.NotificationStreamResponse { return s.send }
func (s *NotificationStream) Send(msg *grpcpb.NotificationStreamResponse) bool {
	select {
	case s.send <- msg:
		return true
	default:
		return false
	}
}
func (s *NotificationStream) Close() {
	s.once.Do(func() { close(s.send) })
}

type NotificationHub struct {
	mu      sync.RWMutex
	clients map[string]map[*NotificationStream]struct{}
}

func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		clients: make(map[string]map[*NotificationStream]struct{}),
	}
}

func (h *NotificationHub) Register(client *NotificationStream) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[client.userID] == nil {
		h.clients[client.userID] = make(map[*NotificationStream]struct{})
	}
	h.clients[client.userID][client] = struct{}{}
}

func (h *NotificationHub) Unregister(client *NotificationStream) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[client.userID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.clients, client.userID)
		}
	}
	client.Close()
}

func (h *NotificationHub) SendToUser(userID string, msg *grpcpb.NotificationStreamResponse) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients[userID] {
		client.Send(msg)
	}
}
