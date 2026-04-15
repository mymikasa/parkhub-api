package grpc

import (
	"context"
	"net"
	"testing"

	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/service"
	servicemocks "github.com/parkhub/api/internal/domains/identity/service/mocks"
	commonv1 "github.com/parkhub/api/internal/gen/api/proto/common/v1"
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

const bufSize = 1024 * 1024

func setupTestServer(t *testing.T) (identityv1.TenantServiceClient, *servicemocks.MockTenantService, *go_mock.Controller) {
	t.Helper()
	ctrl := go_mock.NewController(t)
	mockSvc := servicemocks.NewMockTenantService(ctrl)

	srv := NewTenantGRPCServer(mockSvc)
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	identityv1.RegisterTenantServiceServer(s, srv)
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

	return identityv1.NewTenantServiceClient(conn), mockSvc, ctrl
}

func newDomainTenant() *domain.Tenant {
	return &domain.Tenant{
		ID: "test-id", Name: "Acme", ContactName: "John",
		ContactPhone: "13800138000", ContactEmail: "john@acme.com",
		Status: domain.TenantStatusActive, Address: "123 St",
		PlanType: domain.PlanTypeFree, CreatedAt: 1700000000000, UpdatedAt: 1700000000000,
	}
}

func TestGRPC_CreateTenant_Success(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	expected := newDomainTenant()
	mockSvc.EXPECT().
		CreateTenant(go_mock.Any(), go_mock.Any()).
		Return(expected, nil)

	resp, err := client.CreateTenant(context.Background(), &identityv1.CreateTenantRequest{
		Name: "Acme", ContactName: "John", ContactPhone: "13800138000",
		ContactEmail: "john@acme.com", Address: "123 St", PlanType: identityv1.PlanType_PLAN_TYPE_FREE,
	})
	assert.NoError(t, err)
	assert.Equal(t, "test-id", resp.Tenant.TenantId)
	assert.Equal(t, "Acme", resp.Tenant.Name)
	assert.Equal(t, identityv1.TenantStatus_TENANT_STATUS_ACTIVE, resp.Tenant.Status)
}

func TestGRPC_CreateTenant_AlreadyExists(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		CreateTenant(go_mock.Any(), go_mock.Any()).
		Return(nil, errs.ErrTenantAlreadyExists)

	_, err := client.CreateTenant(context.Background(), &identityv1.CreateTenantRequest{Name: "Dup"})
	assert.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestGRPC_GetTenant_Success(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	expected := newDomainTenant()
	mockSvc.EXPECT().
		GetTenant(go_mock.Any(), "test-id").
		Return(expected, nil)

	resp, err := client.GetTenant(context.Background(), &identityv1.GetTenantRequest{TenantId: "test-id"})
	assert.NoError(t, err)
	assert.Equal(t, "Acme", resp.Tenant.Name)
}

func TestGRPC_GetTenant_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		GetTenant(go_mock.Any(), "bad-id").
		Return(nil, errs.ErrTenantNotFound)

	_, err := client.GetTenant(context.Background(), &identityv1.GetTenantRequest{TenantId: "bad-id"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_ListTenants_Defaults(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		ListTenants(go_mock.Any(), go_mock.Any()).
		DoAndReturn(func(_ context.Context, req *service.ListTenantsRequest) (*service.TenantListResponse, error) {
			assert.Equal(t, 1, req.Page)
			assert.Equal(t, 20, req.PageSize)
			return &service.TenantListResponse{Tenants: []*domain.Tenant{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
		})

	resp, err := client.ListTenants(context.Background(), &identityv1.ListTenantsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.Pagination.Page)
	assert.Equal(t, int32(20), resp.Pagination.PageSize)
}

func TestGRPC_ListTenants_WithPagination(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		ListTenants(go_mock.Any(), go_mock.Any()).
		DoAndReturn(func(_ context.Context, req *service.ListTenantsRequest) (*service.TenantListResponse, error) {
			assert.Equal(t, 2, req.Page)
			assert.Equal(t, 10, req.PageSize)
			return &service.TenantListResponse{
				Tenants: []*domain.Tenant{newDomainTenant()},
				Total:   1, Page: 2, PageSize: 10, TotalPages: 1,
			}, nil
		})

	resp, err := client.ListTenants(context.Background(), &identityv1.ListTenantsRequest{
		Pagination: &commonv1.PaginationRequest{Page: 2, PageSize: 10},
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Tenants, 1)
	assert.Equal(t, int32(2), resp.Pagination.Page)
	assert.Equal(t, int32(10), resp.Pagination.PageSize)
	assert.Equal(t, int64(1), resp.Pagination.Total)
}

func TestGRPC_UpdateTenant_Success(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	updated := newDomainTenant()
	updated.Name = "Updated"
	mockSvc.EXPECT().
		UpdateTenant(go_mock.Any(), go_mock.Any()).
		Return(updated, nil)

	resp, err := client.UpdateTenant(context.Background(), &identityv1.UpdateTenantRequest{
		TenantId: "test-id",
		Name:     strPtr("Updated"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "Updated", resp.Tenant.Name)
}

func TestGRPC_UpdateTenant_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		UpdateTenant(go_mock.Any(), go_mock.Any()).
		Return(nil, errs.ErrTenantNotFound)

	_, err := client.UpdateTenant(context.Background(), &identityv1.UpdateTenantRequest{
		TenantId: "bad-id",
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_FreezeTenant_Success(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	frozen := newDomainTenant()
	frozen.Status = domain.TenantStatusFrozen
	mockSvc.EXPECT().
		FreezeTenant(go_mock.Any(), "test-id").
		Return(frozen, nil)

	resp, err := client.FreezeTenant(context.Background(), &identityv1.FreezeTenantRequest{TenantId: "test-id"})
	assert.NoError(t, err)
	assert.Equal(t, identityv1.TenantStatus_TENANT_STATUS_FROZEN, resp.Tenant.Status)
}

func TestGRPC_FreezeTenant_AlreadyFrozen(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		FreezeTenant(go_mock.Any(), "test-id").
		Return(nil, errs.ErrTenantInvalidStatus)

	_, err := client.FreezeTenant(context.Background(), &identityv1.FreezeTenantRequest{TenantId: "test-id"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_FreezeTenant_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		FreezeTenant(go_mock.Any(), "bad-id").
		Return(nil, errs.ErrTenantNotFound)

	_, err := client.FreezeTenant(context.Background(), &identityv1.FreezeTenantRequest{TenantId: "bad-id"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_UnfreezeTenant_Success(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	active := newDomainTenant()
	mockSvc.EXPECT().
		UnfreezeTenant(go_mock.Any(), "test-id").
		Return(active, nil)

	resp, err := client.UnfreezeTenant(context.Background(), &identityv1.UnfreezeTenantRequest{TenantId: "test-id"})
	assert.NoError(t, err)
	assert.Equal(t, identityv1.TenantStatus_TENANT_STATUS_ACTIVE, resp.Tenant.Status)
}

func TestGRPC_UnfreezeTenant_AlreadyActive(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		UnfreezeTenant(go_mock.Any(), "test-id").
		Return(nil, errs.ErrTenantInvalidStatus)

	_, err := client.UnfreezeTenant(context.Background(), &identityv1.UnfreezeTenantRequest{TenantId: "test-id"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_DeleteTenant_Success(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		DeleteTenant(go_mock.Any(), "test-id").
		Return(nil)

	_, err := client.DeleteTenant(context.Background(), &identityv1.DeleteTenantRequest{TenantId: "test-id"})
	assert.NoError(t, err)
}

func TestGRPC_DeleteTenant_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		DeleteTenant(go_mock.Any(), "bad-id").
		Return(errs.ErrTenantNotFound)

	_, err := client.DeleteTenant(context.Background(), &identityv1.DeleteTenantRequest{TenantId: "bad-id"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestToProtoTenant_Nil(t *testing.T) {
	assert.Nil(t, toProtoTenant(nil))
}

func TestDomainStatusToProto_AllCases(t *testing.T) {
	tests := []struct {
		input    domain.TenantStatus
		expected identityv1.TenantStatus
	}{
		{domain.TenantStatusActive, identityv1.TenantStatus_TENANT_STATUS_ACTIVE},
		{domain.TenantStatusFrozen, identityv1.TenantStatus_TENANT_STATUS_FROZEN},
		{"unknown", identityv1.TenantStatus_TENANT_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, domainStatusToProto(tt.input), "input: %s", tt.input)
	}
}

func TestDomainPlanToProto_AllCases(t *testing.T) {
	tests := []struct {
		input    domain.PlanType
		expected identityv1.PlanType
	}{
		{domain.PlanTypeFree, identityv1.PlanType_PLAN_TYPE_FREE},
		{domain.PlanTypeBasic, identityv1.PlanType_PLAN_TYPE_BASIC},
		{domain.PlanTypePro, identityv1.PlanType_PLAN_TYPE_PRO},
		{domain.PlanTypeEnterprise, identityv1.PlanType_PLAN_TYPE_ENTERPRISE},
		{"unknown", identityv1.PlanType_PLAN_TYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, domainPlanToProto(tt.input), "input: %s", tt.input)
	}
}

func strPtr(s string) *string {
	return &s
}
