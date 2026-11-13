package grpcserver

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"learn-go/pkg/middleware"
)

// StreamAuthInterceptor enforces JWT authentication for streaming RPCs.
func StreamAuthInterceptor(secret string, allowedRoles []string) grpc.StreamServerInterceptor {
	roleSet := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		roleSet[role] = struct{}{}
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeader := ""
		if values := md.Get("authorization"); len(values) > 0 {
			authHeader = values[0]
		}
		if authHeader == "" {
			return status.Error(codes.Unauthenticated, "missing authorization")
		}

		token := strings.TrimSpace(authHeader)
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = strings.TrimSpace(token[7:])
		}
		if token == "" {
			return status.Error(codes.Unauthenticated, "missing bearer token")
		}

		accountID, role, _, err := middleware.ValidateJWT(token, secret)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}
		if accountID == "" {
			return status.Error(codes.Unauthenticated, "invalid token subject")
		}

		if len(roleSet) > 0 {
			if _, ok := roleSet[role]; !ok {
				return status.Error(codes.PermissionDenied, "insufficient role")
			}
		}

		wrapped := &serverStreamWithContext{ServerStream: ss, ctx: WithAccountContext(ctx, accountID, role)}
		return handler(srv, wrapped)
	}
}

type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStreamWithContext) Context() context.Context {
	return s.ctx
}
