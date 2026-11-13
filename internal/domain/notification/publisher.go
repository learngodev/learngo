package notification

// Publisher 将通知实时推送给在线用户。
type Publisher interface {
	Publish(userID string, n *Notification)
}

// Subscriber 订阅某个用户的实时通知；返回的取消函数会关闭通道。
type Subscriber interface {
	Subscribe(userID string) (<-chan *Notification, func())
}
