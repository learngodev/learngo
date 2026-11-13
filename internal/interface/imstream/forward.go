package imstream

import (
	"context"
	"learn-go/internal/interface/grpcpb"
)

// Forward 将实时 Hub 客户端的响应持续转发给出站发送函数。
// 当 ctx 结束、客户端通道关闭或发送返回错误时返回。
func Forward(ctx context.Context, client *StreamClient, send func(*grpcpb.ConversationStreamResponse) error) error {
	if client == nil {
		return nil
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
			if err := send(resp); err != nil {
				return err
			}
		}
	}
}
