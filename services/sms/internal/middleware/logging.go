package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/parkhub/api/pkg/slogutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryLoggingInterceptor(baseLogger *slog.Logger) grpc.UnaryServerInterceptor {
	loggerBase := baseLogger
	if loggerBase == nil {
		loggerBase = slog.Default()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		logger := loggerBase.With(slog.String("grpc_method", info.FullMethod))
		ctx = slogutil.IntoContext(ctx, logger)

		startedAt := time.Now()
		logger.LogAttrs(ctx, slog.LevelInfo, "grpc request started")

		resp, err := handler(ctx, req)

		code := status.Code(err).String()
		if err == nil {
			code = codes.OK.String()
		}

		attrs := []slog.Attr{
			slog.String("grpc_code", code),
			slog.Duration("duration", time.Since(startedAt)),
		}

		if err != nil {
			logger.LogAttrs(ctx, slog.LevelError, "grpc request finished", append(attrs, slog.String("error", err.Error()))...)
			return resp, err
		}

		logger.LogAttrs(ctx, slog.LevelInfo, "grpc request finished", attrs...)
		return resp, nil
	}
}
