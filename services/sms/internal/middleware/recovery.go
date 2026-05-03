package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/parkhub/api/pkg/slogutil"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryRecoveryInterceptor(baseLogger *slog.Logger) grpc.UnaryServerInterceptor {
	loggerBase := baseLogger
	if loggerBase == nil {
		loggerBase = slog.Default()
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			recoveredErr := fmt.Errorf("panic recovered: %v", recovered)
			span := trace.SpanFromContext(ctx)
			span.RecordError(recoveredErr)
			span.SetStatus(otelcodes.Error, recoveredErr.Error())
			span.AddEvent("panic recovered")

			logger := slogutil.FromContext(ctx)
			if logger == slog.Default() {
				logger = loggerBase
			}
			logger = logger.With(slog.String("grpc_method", info.FullMethod))

			logger.LogAttrs(ctx, slog.LevelError, "grpc panic recovered",
				slog.Any("panic", recovered),
				slog.String("grpc_code", codes.Internal.String()),
				slog.String("stack_trace", string(debug.Stack())),
			)

			err = status.Error(codes.Internal, "internal server error")
		}()

		return handler(ctx, req)
	}
}
