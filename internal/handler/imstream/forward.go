package imstream

import (
	"context"

	"learn-go/api/grpcpb"
	"learn-go/internal/realtime"
)

// Forward pumps responses from the realtime hub client to an outbound send function.
// It returns when ctx is done, client channel closes, or send returns error.
func Forward(ctx context.Context, client *realtime.StreamClient, send func(*grpcpb.ConversationStreamResponse) error) error {
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
