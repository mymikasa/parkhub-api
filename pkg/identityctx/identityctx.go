package identityctx

import "context"

type contextKey string

const (
	userIDKey   contextKey = "x-user-id"
	tenantIDKey contextKey = "x-tenant-id"
	userRoleKey contextKey = "x-user-role"
)

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, userRoleKey, role)
}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

func TenantID(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

func Role(ctx context.Context) string {
	v, _ := ctx.Value(userRoleKey).(string)
	return v
}
