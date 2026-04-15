package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/service"
	servicemocks "github.com/parkhub/api/internal/domains/identity/service/mocks"
	identityv1 "github.com/parkhub/api/internal/gen/api/proto/identity/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	go_mock "go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func setupAuthTestServer(t *testing.T) (identityv1.AuthServiceClient, *servicemocks.MockAuthService, *go_mock.Controller) {
	t.Helper()
	ctrl := go_mock.NewController(t)
	mockSvc := servicemocks.NewMockAuthService(ctrl)

	srv := NewAuthGRPCServer(mockSvc, nil)
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	identityv1.RegisterAuthServiceServer(s, srv)
	go s.Serve(lis)
	t.Cleanup(s.GracefulStop)

	conn, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return identityv1.NewAuthServiceClient(conn), mockSvc, ctrl
}

func newAuthDomainUser() *domain.User {
	tenantID := "tenant-1"
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return &domain.User{
		ID:        "user-1",
		TenantID:  &tenantID,
		Username:  "admin",
		RealName:  "Admin",
		Role:      domain.RoleTenantAdmin,
		Status:    domain.UserStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestGRPCAuth_Login_Success(t *testing.T) {
	client, mockSvc, ctrl := setupAuthTestServer(t)
	defer ctrl.Finish()

	user := newAuthDomainUser()
	mockSvc.EXPECT().Login(go_mock.Any(), go_mock.Any()).Return(&service.LoginResponse{
		TokenPair: domain.TokenPair{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		},
		User:            user,
		AccessExpiresIn: 900,
	}, nil)

	resp, err := client.Login(context.Background(), &identityv1.LoginRequest{
		Username: "admin",
		Password: "password123",
	})
	assert.NoError(t, err)
	assert.Equal(t, "access-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
	assert.Equal(t, int32(900), resp.AccessExpiresIn)
	assert.Equal(t, "Bearer", resp.TokenType)
	assert.Equal(t, "user-1", resp.User.UserId)
}

func TestGRPCAuth_Login_InvalidCredentials(t *testing.T) {
	client, mockSvc, ctrl := setupAuthTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().Login(go_mock.Any(), go_mock.Any()).Return(nil, errs.ErrInvalidCredentials)

	_, err := client.Login(context.Background(), &identityv1.LoginRequest{
		Username: "admin",
		Password: "wrong",
	})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPCAuth_Login_FrozenUser(t *testing.T) {
	client, mockSvc, ctrl := setupAuthTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().Login(go_mock.Any(), go_mock.Any()).Return(nil, errs.ErrUserFrozen)

	_, err := client.Login(context.Background(), &identityv1.LoginRequest{
		Username: "admin",
		Password: "password123",
	})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPCAuth_RefreshToken_Success(t *testing.T) {
	client, mockSvc, ctrl := setupAuthTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().RefreshToken(go_mock.Any(), go_mock.Any()).Return(&service.RefreshTokenResponse{
		TokenPair: domain.TokenPair{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
		},
		AccessExpiresIn: 900,
	}, nil)

	resp, err := client.RefreshToken(context.Background(), &identityv1.RefreshTokenRequest{
		RefreshToken: "old-refresh-token",
	})
	assert.NoError(t, err)
	assert.Equal(t, "new-access", resp.AccessToken)
	assert.Equal(t, "new-refresh", resp.RefreshToken)
	assert.Equal(t, "Bearer", resp.TokenType)
}

func TestGRPCAuth_RefreshToken_Invalid(t *testing.T) {
	client, mockSvc, ctrl := setupAuthTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().RefreshToken(go_mock.Any(), go_mock.Any()).Return(nil, errs.ErrRefreshTokenInvalid)

	_, err := client.RefreshToken(context.Background(), &identityv1.RefreshTokenRequest{
		RefreshToken: "bad-token",
	})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPCAuth_RefreshToken_Revoked(t *testing.T) {
	client, mockSvc, ctrl := setupAuthTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().RefreshToken(go_mock.Any(), go_mock.Any()).Return(nil, errs.ErrRefreshTokenRevoked)

	_, err := client.RefreshToken(context.Background(), &identityv1.RefreshTokenRequest{
		RefreshToken: "revoked-token",
	})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPCAuth_Logout_Success(t *testing.T) {
	client, mockSvc, ctrl := setupAuthTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().Logout(go_mock.Any(), go_mock.Any()).Return(nil)

	resp, err := client.Logout(context.Background(), &identityv1.LogoutRequest{
		RefreshToken: "refresh-token",
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestGRPCAuth_Logout_InvalidToken(t *testing.T) {
	client, mockSvc, ctrl := setupAuthTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().Logout(go_mock.Any(), go_mock.Any()).Return(errs.ErrRefreshTokenInvalid)

	_, err := client.Logout(context.Background(), &identityv1.LogoutRequest{
		RefreshToken: "bad-token",
	})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPCAuth_GetJWKS_Success(t *testing.T) {
	client, _, ctrl := setupAuthTestServer(t)
	defer ctrl.Finish()

	resp, err := client.GetJWKS(context.Background(), &identityv1.GetJWKSRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}
