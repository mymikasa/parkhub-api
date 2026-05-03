package middleware

import (
	"context"
	"github.com/parkhub/api/pkg/identityctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var authWhitelist = map[string]struct{}{
	"/parkhub.identity.v1.AuthService/Login":        {},
	"/parkhub.identity.v1.AuthService/SmsLogin":     {},
	"/parkhub.identity.v1.AuthService/RefreshToken": {},
	"/parkhub.identity.v1.AuthService/Logout":       {},
	"/parkhub.identity.v1.AuthService/GetJWKS":      {},
	"/grpc.health.v1.Health/Check":                  {},
	"/grpc.health.v1.Health/Watch":                  {},
}

func UnaryAuthContextInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := authWhitelist[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		userIDs := md.Get("x-user-id")
		if len(userIDs) == 0 || userIDs[0] == "" {
			return nil, status.Error(codes.Unauthenticated, "missing x-user-id")
		}

		ctx = identityctx.WithUserID(ctx, userIDs[0])

		tenantIDs := md.Get("x-tenant-id")
		if len(tenantIDs) > 0 {
			ctx = identityctx.WithTenantID(ctx, tenantIDs[0])
		}

		roles := md.Get("x-user-role")
		if len(roles) > 0 {
			ctx = identityctx.WithRole(ctx, roles[0])
		}

		return handler(ctx, req)
	}
}
