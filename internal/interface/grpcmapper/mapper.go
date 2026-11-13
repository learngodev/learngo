package grpcmapper

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"learn-go/internal/domain/messaging"
	"learn-go/internal/interface/grpcpb"
	"learn-go/internal/usecase"
)

// Message 将领域消息转换为 protobuf 表示。
func Message(msg messaging.Message) *grpcpb.Message {
	return &grpcpb.Message{
		Id:             msg.ID,
		ConversationId: msg.ConversationID,
		SenderId:       msg.SenderID,
		SenderRole:     string(msg.SenderRole),
		Kind:           msg.Kind,
		Text:           msg.Text,
		MediaUri:       msg.MediaURI,
		Metadata:       msg.Metadata,
		CreatedAt:      timestamppb.New(msg.CreatedAt),
	}
}

// ConversationSummary 将用例层的会话摘要转换为 protobuf。
func ConversationSummary(summary usecase.ConversationSummary) *grpcpb.ConversationSummary {
	members := make([]*grpcpb.ConversationMember, 0, len(summary.Members))
	for _, member := range summary.Members {
		members = append(members, &grpcpb.ConversationMember{
			Id:             member.ID,
			ConversationId: member.ConversationID,
			AccountId:      member.AccountID,
			Role:           string(member.Role),
			CreatedAt:      timestamppb.New(member.CreatedAt),
		})
	}

	var lastMessage *grpcpb.Message
	if summary.LastMessage != nil {
		lastMessage = Message(*summary.LastMessage)
	}

	return &grpcpb.ConversationSummary{
		Conversation: &grpcpb.Conversation{
			Id:        summary.Conversation.ID,
			Type:      summary.Conversation.Type,
			SchoolId:  summary.Conversation.SchoolID,
			CreatedAt: timestamppb.New(summary.Conversation.CreatedAt),
			UpdatedAt: timestamppb.New(summary.Conversation.UpdatedAt),
		},
		Members:     members,
		LastMessage: lastMessage,
		UnreadCount: summary.UnreadCount,
	}
}

// Snapshot 组合会话摘要与消息，生成会话快照。
func Snapshot(summary usecase.ConversationSummary, messages []messaging.Message) *grpcpb.ConversationSnapshot {
	messagePayloads := make([]*grpcpb.Message, 0, len(messages))
	for _, msg := range messages {
		messagePayloads = append(messagePayloads, Message(msg))
	}

	return &grpcpb.ConversationSnapshot{
		Summary:  ConversationSummary(summary),
		Messages: messagePayloads,
	}
}
