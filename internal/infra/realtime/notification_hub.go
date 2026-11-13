package realtime

import (
	"sync"

	"learn-go/internal/domain/notification"
)

type subscription struct {
	userID string
	ch     chan *notification.Notification
	once   sync.Once
}

func (s *subscription) close() {
	s.once.Do(func() { close(s.ch) })
}

// NotificationHub 基于内存实现 notification.Publisher 与 notification.Subscriber。
type NotificationHub struct {
	mu   sync.RWMutex
	subs map[string]map[*subscription]struct{}
}

// NewNotificationHub 构造一个空的 NotificationHub。
func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		subs: make(map[string]map[*subscription]struct{}),
	}
}

// Subscribe 为用户注册一个订阅，返回通知通道及取消订阅函数。
func (h *NotificationHub) Subscribe(userID string) (<-chan *notification.Notification, func()) {
	sub := &subscription{userID: userID, ch: make(chan *notification.Notification, 32)}
	h.mu.Lock()
	if h.subs[userID] == nil {
		h.subs[userID] = make(map[*subscription]struct{})
	}
	h.subs[userID][sub] = struct{}{}
	h.mu.Unlock()
	return sub.ch, func() { h.unsubscribe(sub) }
}

func (h *NotificationHub) unsubscribe(sub *subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.subs[sub.userID]; ok {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(h.subs, sub.userID)
		}
	}
	sub.close()
}

// Publish 将通知推送给用户的所有在线订阅；缓冲区已满的订阅会被跳过。
func (h *NotificationHub) Publish(userID string, n *notification.Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for sub := range h.subs[userID] {
		select {
		case sub.ch <- n:
		default:
		}
	}
}

var (
	_ notification.Publisher  = (*NotificationHub)(nil)
	_ notification.Subscriber = (*NotificationHub)(nil)
)
