package grpcserver

import "context"

type contextKey string

const (
	accountIDKey contextKey = "accountID"
	roleKey      contextKey = "role"
)

// WithAccountContext embeds account metadata into the context.
func WithAccountContext(ctx context.Context, accountID, role string) context.Context {
	ctx = context.WithValue(ctx, accountIDKey, accountID)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
}

// AccountIDFromContext extracts the account identifier from context.
func AccountIDFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value(accountIDKey)
	id, ok := val.(string)
	return id, ok
}

// RoleFromContext extracts the role from context.
func RoleFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value(roleKey)
	role, ok := val.(string)
	return role, ok
}
