package grpcserver

import "context"

type contextKey string

const (
	accountIDKey contextKey = "accountID"
	roleKey      contextKey = "role"
)

// WithAccountContext 将账号信息写入 context。
func WithAccountContext(ctx context.Context, accountID, role string) context.Context {
	ctx = context.WithValue(ctx, accountIDKey, accountID)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
}

// AccountIDFromContext 从 context 中提取账号 ID。
func AccountIDFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value(accountIDKey)
	id, ok := val.(string)
	return id, ok
}

// RoleFromContext 从 context 中提取角色。
func RoleFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value(roleKey)
	role, ok := val.(string)
	return role, ok
}
