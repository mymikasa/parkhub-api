package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/parkhub/api/pkg/identityctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		attrs := []slog.Attr{
			slog.String("method", info.FullMethod),
			slog.Duration("duration", duration),
		}
		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
			logger.LogAttrs(ctx, slog.LevelError, "gRPC call failed", attrs...)
		} else {
			logger.LogAttrs(ctx, slog.LevelInfo, "gRPC call completed", attrs...)
		}

		return resp, err
	}
}

func UnaryAuthContextInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		userIDs := md.Get("x-user-id")
		if len(userIDs) > 0 && userIDs[0] != "" {
			ctx = identityctx.WithUserID(ctx, userIDs[0])
		}

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

func UnaryRecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					slog.String("method", info.FullMethod),
					slog.Any("panic", r),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}
