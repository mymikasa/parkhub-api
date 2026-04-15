package middleware

import (
	"context"
	"net"
	"testing"

	"github.com/parkhub/api/pkg/identityctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func setupAuthContextTestServer(t *testing.T) (context.Context, func()) {
	t.Helper()
	lis := bufconn.Listen(1024 * 32)

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(UnaryAuthContextInterceptor()),
	)
	go s.Serve(lis)

	conn, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	return context.Background(), func() {
		conn.Close()
		s.GracefulStop()
	}
}

func TestUnaryAuthContextInterceptor_WhitelistedMethod(t *testing.T) {
	var capturedCtx context.Context
	invoked := false

	interceptor := UnaryAuthContextInterceptor()
	_, err := interceptor(context.Background(), "req", &grpc.UnaryServerInfo{
		FullMethod: "/parkhub.identity.v1.AuthService/Login",
	}, func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		invoked = true
		return "ok", nil
	})
	assert.NoError(t, err)
	assert.True(t, invoked)
	assert.Equal(t, "", identityctx.UserID(capturedCtx))
}

func TestUnaryAuthContextInterceptor_WhitelistedSmsMethods(t *testing.T) {
	interceptor := UnaryAuthContextInterceptor()

	for _, method := range []string{
		"/parkhub.sms.v1.SmsService/SendCode",
		"/parkhub.sms.v1.SmsService/VerifyCode",
	} {
		t.Run(method, func(t *testing.T) {
			invoked := false

			_, err := interceptor(context.Background(), "req", &grpc.UnaryServerInfo{
				FullMethod: method,
			}, func(ctx context.Context, req any) (any, error) {
				invoked = true
				return "ok", nil
			})

			assert.NoError(t, err)
			assert.True(t, invoked)
		})
	}
}

func TestUnaryAuthContextInterceptor_ProtectedWithHeaders(t *testing.T) {
	var capturedCtx context.Context
	interceptor := UnaryAuthContextInterceptor()

	md := metadata.Pairs("x-user-id", "user-123", "x-tenant-id", "tenant-456", "x-user-role", "admin")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{
		FullMethod: "/parkhub.identity.v1.UserService/GetUser",
	}, func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "ok", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "user-123", identityctx.UserID(capturedCtx))
	assert.Equal(t, "tenant-456", identityctx.TenantID(capturedCtx))
	assert.Equal(t, "admin", identityctx.Role(capturedCtx))
}

func TestUnaryAuthContextInterceptor_ProtectedWithoutHeaders(t *testing.T) {
	interceptor := UnaryAuthContextInterceptor()

	_, err := interceptor(context.Background(), "req", &grpc.UnaryServerInfo{
		FullMethod: "/parkhub.identity.v1.UserService/GetUser",
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryAuthContextInterceptor_PartialHeaders(t *testing.T) {
	var capturedCtx context.Context
	interceptor := UnaryAuthContextInterceptor()

	md := metadata.Pairs("x-user-id", "user-123")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{
		FullMethod: "/parkhub.identity.v1.UserService/GetUser",
	}, func(ctx context.Context, req any) (any, error) {
		capturedCtx = ctx
		return "ok", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, "user-123", identityctx.UserID(capturedCtx))
	assert.Equal(t, "", identityctx.TenantID(capturedCtx))
	assert.Equal(t, "", identityctx.Role(capturedCtx))
}

func TestUnaryAuthContextInterceptor_HealthCheck(t *testing.T) {
	interceptor := UnaryAuthContextInterceptor()

	_, err := interceptor(context.Background(), "req", &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	assert.NoError(t, err)
}
