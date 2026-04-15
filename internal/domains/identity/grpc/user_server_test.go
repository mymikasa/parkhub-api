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
	commonv1 "github.com/parkhub/api/internal/gen/api/proto/common/v1"
	identityv1 "github.com/parkhub/api/internal/gen/api/proto/identity/v1"
	"github.com/parkhub/api/pkg/identityctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	go_mock "go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func setupUserTestServer(t *testing.T) (identityv1.UserServiceClient, *servicemocks.MockUserService, *go_mock.Controller) {
	t.Helper()
	ctrl := go_mock.NewController(t)
	mockSvc := servicemocks.NewMockUserService(ctrl)

	injectIdentity := func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if ids := md.Get("x-user-id"); len(ids) > 0 {
				ctx = identityctx.WithUserID(ctx, ids[0])
			}
		}
		return handler(ctx, req)
	}

	srv := NewUserGRPCServer(mockSvc)
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer(grpc.UnaryInterceptor(injectIdentity))
	identityv1.RegisterUserServiceServer(s, srv)
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

	return identityv1.NewUserServiceClient(conn), mockSvc, ctrl
}

func newDomainUser() *domain.User {
	tenantID := "tenant-1"
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return &domain.User{
		ID:        "user-1",
		TenantID:  &tenantID,
		Username:  "alice",
		RealName:  "Alice",
		Role:      domain.RoleTenantAdmin,
		Status:    domain.UserStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestGRPC_CreateUser_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	expected := newDomainUser()
	mockSvc.EXPECT().Create(go_mock.Any(), go_mock.Any()).Return(expected, nil)

	tenantID := "tenant-1"
	resp, err := client.CreateUser(context.Background(), &identityv1.CreateUserRequest{
		TenantId: &tenantID,
		Username: "alice",
		Password: "password123",
		RealName: "Alice",
		Role:     identityv1.UserRole_USER_ROLE_TENANT_ADMIN,
	})
	assert.NoError(t, err)
	assert.Equal(t, "user-1", resp.User.UserId)
	assert.Equal(t, "alice", resp.User.Username)
	assert.Equal(t, identityv1.UserRole_USER_ROLE_TENANT_ADMIN, resp.User.Role)
}

func TestGRPC_CreateUser_AlreadyExists(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().Create(go_mock.Any(), go_mock.Any()).Return(nil, errs.ErrUserAlreadyExists)

	_, err := client.CreateUser(context.Background(), &identityv1.CreateUserRequest{
		Username: "dup",
		Password: "password123",
		RealName: "Dup",
		Role:     identityv1.UserRole_USER_ROLE_OPERATOR,
	})
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestGRPC_GetUser_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	expected := newDomainUser()
	mockSvc.EXPECT().GetByID(go_mock.Any(), go_mock.Any()).Return(expected, nil)

	resp, err := client.GetUser(context.Background(), &identityv1.GetUserRequest{UserId: "user-1"})
	assert.NoError(t, err)
	assert.Equal(t, "alice", resp.User.Username)
}

func TestGRPC_GetUser_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().GetByID(go_mock.Any(), go_mock.Any()).Return(nil, errs.ErrUserNotFound)

	_, err := client.GetUser(context.Background(), &identityv1.GetUserRequest{UserId: "bad-id"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_ListUsers_Defaults(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().List(go_mock.Any(), go_mock.Any()).
		DoAndReturn(func(_ context.Context, req *service.ListUsersRequest) (*service.UserListResponse, error) {
			assert.Equal(t, 1, req.Page)
			assert.Equal(t, 20, req.PageSize)
			return &service.UserListResponse{Users: []*domain.User{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
		})

	resp, err := client.ListUsers(context.Background(), &identityv1.ListUsersRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.Pagination.Page)
	assert.Equal(t, int32(20), resp.Pagination.PageSize)
}

func TestGRPC_ListUsers_WithPagination(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().List(go_mock.Any(), go_mock.Any()).
		DoAndReturn(func(_ context.Context, req *service.ListUsersRequest) (*service.UserListResponse, error) {
			assert.Equal(t, 2, req.Page)
			assert.Equal(t, 10, req.PageSize)
			return &service.UserListResponse{
				Users: []*domain.User{newDomainUser()},
				Total: 1, Page: 2, PageSize: 10, TotalPages: 1,
			}, nil
		})

	resp, err := client.ListUsers(context.Background(), &identityv1.ListUsersRequest{
		Pagination: &commonv1.PaginationRequest{Page: 2, PageSize: 10},
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Users, 1)
	assert.Equal(t, int32(2), resp.Pagination.Page)
}

func TestGRPC_UpdateUser_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	updated := newDomainUser()
	updated.RealName = "Updated"
	mockSvc.EXPECT().Update(go_mock.Any(), go_mock.Any()).Return(updated, nil)

	newName := "Updated"
	resp, err := client.UpdateUser(context.Background(), &identityv1.UpdateUserRequest{
		UserId:   "user-1",
		RealName: &newName,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Updated", resp.User.RealName)
}

func TestGRPC_FreezeUser_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().Freeze(go_mock.Any(), go_mock.Any()).Return(nil)

	_, err := client.FreezeUser(context.Background(), &identityv1.FreezeUserRequest{UserId: "user-1"})
	assert.NoError(t, err)
}

func TestGRPC_FreezeUser_InvalidStatus(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().Freeze(go_mock.Any(), go_mock.Any()).Return(errs.ErrUserInvalidStatus)

	_, err := client.FreezeUser(context.Background(), &identityv1.FreezeUserRequest{UserId: "user-1"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_UnfreezeUser_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().Unfreeze(go_mock.Any(), go_mock.Any()).Return(nil)

	_, err := client.UnfreezeUser(context.Background(), &identityv1.UnfreezeUserRequest{UserId: "user-1"})
	assert.NoError(t, err)
}

func TestGRPC_ResetPassword_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().ResetPassword(go_mock.Any(), go_mock.Any()).Return(nil)

	_, err := client.ResetPassword(context.Background(), &identityv1.ResetPasswordRequest{
		UserId:      "user-1",
		NewPassword: "newpass",
	})
	assert.NoError(t, err)
}

func TestGRPC_UpdateProfile_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	updated := newDomainUser()
	updated.RealName = "New Name"
	mockSvc.EXPECT().UpdateProfile(go_mock.Any(), "user-1", go_mock.Any()).Return(updated, nil)

	newName := "New Name"
	resp, err := client.UpdateProfile(context.Background(), &identityv1.UpdateProfileRequest{
		UserId:   "user-1",
		RealName: &newName,
	})
	assert.NoError(t, err)
	assert.Equal(t, "New Name", resp.User.RealName)
}

func TestGRPC_ChangePassword_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().ChangePassword(go_mock.Any(), "user-1", go_mock.Any()).Return(nil)

	_, err := client.ChangePassword(context.Background(), &identityv1.ChangePasswordRequest{
		UserId:      "user-1",
		OldPassword: "old",
		NewPassword: "new",
	})
	assert.NoError(t, err)
}

func TestGRPC_ChangePassword_IncorrectPassword(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().ChangePassword(go_mock.Any(), "user-1", go_mock.Any()).Return(errs.ErrPasswordIncorrect)

	_, err := client.ChangePassword(context.Background(), &identityv1.ChangePasswordRequest{
		UserId:      "user-1",
		OldPassword: "wrong",
		NewPassword: "new",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_ImportUsers_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().ImportUsers(go_mock.Any(), go_mock.Any()).Return(&service.ImportResult{
		Total:   2,
		Success: 1,
		Failed:  1,
		Errors:  []service.ImportError{{Index: 1, Message: "duplicate"}},
	}, nil)

	resp, err := client.ImportUsers(context.Background(), &identityv1.ImportUsersRequest{
		Users: []*identityv1.ImportUserItem{
			{Username: "alice", Password: "pass1", RealName: "Alice", Role: identityv1.UserRole_USER_ROLE_OPERATOR},
			{Username: "bob", Password: "pass2", RealName: "Bob", Role: identityv1.UserRole_USER_ROLE_OPERATOR},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.Total)
	assert.Equal(t, int32(1), resp.Success)
	assert.Equal(t, int32(1), resp.Failed)
	assert.Len(t, resp.Errors, 1)
}

func TestGRPC_GetCurrentUser_Success(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	expected := newDomainUser()
	mockSvc.EXPECT().GetByID(go_mock.Any(), &service.GetUserRequest{UserID: "user-1"}).Return(expected, nil)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-user-id", "user-1")
	resp, err := client.GetCurrentUser(ctx, &identityv1.GetCurrentUserRequest{})
	assert.NoError(t, err)
	assert.Equal(t, "user-1", resp.User.UserId)
	assert.Equal(t, "alice", resp.User.Username)
}

func TestGRPC_GetCurrentUser_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupUserTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().GetByID(go_mock.Any(), &service.GetUserRequest{UserID: "user-1"}).Return(nil, errs.ErrUserNotFound)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "x-user-id", "user-1")
	_, err := client.GetCurrentUser(ctx, &identityv1.GetCurrentUserRequest{})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestToProtoUser_Nil(t *testing.T) {
	assert.Nil(t, toProtoUser(nil))
}

func TestDomainRoleToProto_AllCases(t *testing.T) {
	tests := []struct {
		input    domain.UserRole
		expected identityv1.UserRole
	}{
		{domain.RolePlatformAdmin, identityv1.UserRole_USER_ROLE_PLATFORM_ADMIN},
		{domain.RoleTenantAdmin, identityv1.UserRole_USER_ROLE_TENANT_ADMIN},
		{domain.RoleOperator, identityv1.UserRole_USER_ROLE_OPERATOR},
		{"unknown", identityv1.UserRole_USER_ROLE_UNSPECIFIED},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, domainRoleToProto(tt.input), "input: %s", tt.input)
	}
}

func TestDomainUserStatusToProto_AllCases(t *testing.T) {
	tests := []struct {
		input    domain.UserStatus
		expected identityv1.UserStatus
	}{
		{domain.UserStatusActive, identityv1.UserStatus_USER_STATUS_ACTIVE},
		{domain.UserStatusFrozen, identityv1.UserStatus_USER_STATUS_FROZEN},
		{"unknown", identityv1.UserStatus_USER_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, domainUserStatusToProto(tt.input), "input: %s", tt.input)
	}
}
