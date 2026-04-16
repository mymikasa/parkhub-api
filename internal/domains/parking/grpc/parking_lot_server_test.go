package grpc

import (
	"context"
	"net"
	"testing"

	"github.com/parkhub/api/internal/domains/parking/domain"
	"github.com/parkhub/api/internal/domains/parking/errs"
	"github.com/parkhub/api/internal/domains/parking/service"
	servicemocks "github.com/parkhub/api/internal/domains/parking/service/mocks"
	commonv1 "github.com/parkhub/api/internal/gen/api/proto/common/v1"
	parkingv1 "github.com/parkhub/api/internal/gen/api/proto/parking/v1"
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

func setupParkingTestServer(t *testing.T) (parkingv1.ParkingLotServiceClient, *servicemocks.MockParkingLotService, *go_mock.Controller) {
	t.Helper()
	ctrl := go_mock.NewController(t)
	mockSvc := servicemocks.NewMockParkingLotService(ctrl)

	srv := NewParkingLotGRPCServer(mockSvc)
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer(
		grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				if tids := md.Get("x-tenant-id"); len(tids) > 0 {
					ctx = identityctx.WithTenantID(ctx, tids[0])
				}
				if uids := md.Get("x-user-id"); len(uids) > 0 {
					ctx = identityctx.WithUserID(ctx, uids[0])
				}
			}
			return handler(ctx, req)
		}),
	)
	parkingv1.RegisterParkingLotServiceServer(s, srv)
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

	return parkingv1.NewParkingLotServiceClient(conn), mockSvc, ctrl
}

func ctxWithTenant() context.Context {
	ctx := context.Background()
	md := metadata.Pairs("x-tenant-id", "tenant-1", "x-user-id", "user-1")
	return metadata.NewOutgoingContext(ctx, md)
}

func newDomainParkingLot() *domain.ParkingLot {
	return &domain.ParkingLot{
		ID:              "lot-1",
		TenantID:        "tenant-1",
		Name:            "朝阳停车场",
		Address:         "朝阳区xxx",
		TotalSpaces:     200,
		AvailableSpaces: 180,
		LotType:         domain.LotTypeUnderground,
		Status:          domain.ParkingLotStatusActive,
		CreatedAt:       1700000000000,
		UpdatedAt:       1700000000000,
	}
}

func TestGRPC_CreateParkingLot_Success(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	expected := newDomainParkingLot()
	mockSvc.EXPECT().
		Create(go_mock.Any(), go_mock.Any()).
		Return(expected, nil)

	resp, err := client.CreateParkingLot(ctxWithTenant(), &parkingv1.CreateParkingLotRequest{
		Name: "朝阳停车场", Address: "朝阳区xxx", TotalSpaces: 200, LotType: parkingv1.LotType_LOT_TYPE_UNDERGROUND,
	})
	assert.NoError(t, err)
	assert.Equal(t, "lot-1", resp.ParkingLot.Id)
	assert.Equal(t, "tenant-1", resp.ParkingLot.TenantId)
	assert.Equal(t, "朝阳停车场", resp.ParkingLot.Name)
	assert.Equal(t, int32(200), resp.ParkingLot.TotalSpaces)
	assert.Equal(t, int32(180), resp.ParkingLot.AvailableSpaces)
	assert.Equal(t, parkingv1.LotType_LOT_TYPE_UNDERGROUND, resp.ParkingLot.LotType)
	assert.Equal(t, parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_ACTIVE, resp.ParkingLot.Status)
}

func TestGRPC_CreateParkingLot_MissingTenantID(t *testing.T) {
	client, _, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	_, err := client.CreateParkingLot(context.Background(), &parkingv1.CreateParkingLotRequest{
		Name: "test", Address: "addr", TotalSpaces: 100,
	})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPC_CreateParkingLot_MissingName(t *testing.T) {
	client, _, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	_, err := client.CreateParkingLot(ctxWithTenant(), &parkingv1.CreateParkingLotRequest{
		Address: "addr", TotalSpaces: 100,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_CreateParkingLot_MissingAddress(t *testing.T) {
	client, _, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	_, err := client.CreateParkingLot(ctxWithTenant(), &parkingv1.CreateParkingLotRequest{
		Name: "test", TotalSpaces: 100,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_CreateParkingLot_ZeroTotalSpaces(t *testing.T) {
	client, _, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	_, err := client.CreateParkingLot(ctxWithTenant(), &parkingv1.CreateParkingLotRequest{
		Name: "test", Address: "addr", TotalSpaces: 0,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_CreateParkingLot_DuplicateName(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Create(go_mock.Any(), go_mock.Any()).
		Return(nil, errs.ErrParkingLotNameDuplicate)

	_, err := client.CreateParkingLot(ctxWithTenant(), &parkingv1.CreateParkingLotRequest{
		Name: "dup", Address: "addr", TotalSpaces: 100,
	})
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestGRPC_GetParkingLot_Success(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	expected := newDomainParkingLot()
	mockSvc.EXPECT().
		GetByID(go_mock.Any(), "tenant-1", "lot-1").
		Return(expected, nil)

	resp, err := client.GetParkingLot(ctxWithTenant(), &parkingv1.GetParkingLotRequest{Id: "lot-1"})
	assert.NoError(t, err)
	assert.Equal(t, "朝阳停车场", resp.ParkingLot.Name)
}

func TestGRPC_GetParkingLot_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		GetByID(go_mock.Any(), "tenant-1", "bad-id").
		Return(nil, errs.ErrParkingLotNotFound)

	_, err := client.GetParkingLot(ctxWithTenant(), &parkingv1.GetParkingLotRequest{Id: "bad-id"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_ListParkingLots_WithPagination(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		List(go_mock.Any(), go_mock.Any()).
		DoAndReturn(func(_ context.Context, req *service.ListParkingLotsRequest) (*service.ParkingLotListResponse, error) {
			assert.Equal(t, "tenant-1", req.TenantID)
			assert.Equal(t, 2, req.Page)
			assert.Equal(t, 10, req.PageSize)
			return &service.ParkingLotListResponse{
				ParkingLots: []*domain.ParkingLot{newDomainParkingLot()},
				Total:       1, Page: 2, PageSize: 10, TotalPages: 1,
			}, nil
		})

	resp, err := client.ListParkingLots(ctxWithTenant(), &parkingv1.ListParkingLotsRequest{
		Pagination: &commonv1.PaginationRequest{Page: 2, PageSize: 10},
	})
	assert.NoError(t, err)
	assert.Len(t, resp.ParkingLots, 1)
	assert.Equal(t, int32(2), resp.Pagination.Page)
	assert.Equal(t, int32(10), resp.Pagination.PageSize)
}

func TestGRPC_ListParkingLots_ReturnsServicePagination(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		List(go_mock.Any(), go_mock.Any()).
		Return(&service.ParkingLotListResponse{
			ParkingLots: []*domain.ParkingLot{},
			Total:       50, Page: 1, PageSize: 100, TotalPages: 1,
		}, nil)

	resp, err := client.ListParkingLots(ctxWithTenant(), &parkingv1.ListParkingLotsRequest{
		Pagination: &commonv1.PaginationRequest{Page: 1, PageSize: 999},
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(100), resp.Pagination.PageSize)
}

func TestGRPC_ListParkingLots_UnspecifiedFiltersNoFilter(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		List(go_mock.Any(), go_mock.Any()).
		DoAndReturn(func(_ context.Context, req *service.ListParkingLotsRequest) (*service.ParkingLotListResponse, error) {
			assert.Equal(t, domain.ParkingLotStatus(""), req.Status)
			assert.Equal(t, domain.LotType(""), req.LotType)
			return &service.ParkingLotListResponse{}, nil
		})

	_, err := client.ListParkingLots(ctxWithTenant(), &parkingv1.ListParkingLotsRequest{})
	assert.NoError(t, err)
}

func TestGRPC_UpdateParkingLot_PartialUpdate(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	updated := newDomainParkingLot()
	updated.Name = "新名字"
	mockSvc.EXPECT().
		Update(go_mock.Any(), go_mock.Any()).
		Return(updated, nil)

	resp, err := client.UpdateParkingLot(ctxWithTenant(), &parkingv1.UpdateParkingLotRequest{
		Id:   "lot-1",
		Name: strPtr("新名字"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "新名字", resp.ParkingLot.Name)
}

func TestGRPC_UpdateParkingLot_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Update(go_mock.Any(), go_mock.Any()).
		Return(nil, errs.ErrParkingLotNotFound)

	_, err := client.UpdateParkingLot(ctxWithTenant(), &parkingv1.UpdateParkingLotRequest{
		Id: "bad-id",
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_UpdateParkingLot_InvalidStatus(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Update(go_mock.Any(), go_mock.Any()).
		Return(nil, errs.ErrParkingLotInvalidStatus)

	_, err := client.UpdateParkingLot(ctxWithTenant(), &parkingv1.UpdateParkingLotRequest{
		Id:     "lot-1",
		Status: parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_INACTIVE.Enum(),
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_DeleteParkingLot_Success(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Delete(go_mock.Any(), "tenant-1", "lot-1").
		Return(nil)

	_, err := client.DeleteParkingLot(ctxWithTenant(), &parkingv1.DeleteParkingLotRequest{Id: "lot-1"})
	assert.NoError(t, err)
}

func TestGRPC_DeleteParkingLot_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Delete(go_mock.Any(), "tenant-1", "bad-id").
		Return(errs.ErrParkingLotNotFound)

	_, err := client.DeleteParkingLot(ctxWithTenant(), &parkingv1.DeleteParkingLotRequest{Id: "bad-id"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_GetParkingLotStats_Success(t *testing.T) {
	client, mockSvc, ctrl := setupParkingTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		GetStats(go_mock.Any(), "tenant-1").
		Return(&service.ParkingLotStatsResponse{
			TotalSpaces: 300, AvailableSpaces: 230, OccupiedVehicles: 70, TotalGates: 0,
		}, nil)

	resp, err := client.GetParkingLotStats(ctxWithTenant(), &parkingv1.GetParkingLotStatsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int64(300), resp.TotalSpaces)
	assert.Equal(t, int64(230), resp.AvailableSpaces)
	assert.Equal(t, int64(70), resp.OccupiedVehicles)
	assert.Equal(t, int64(0), resp.TotalGates)
}

func TestToProtoParkingLot_Nil(t *testing.T) {
	assert.Nil(t, toProtoParkingLot(nil))
}

func TestDomainLotTypeToProto_AllCases(t *testing.T) {
	tests := []struct {
		input    domain.LotType
		expected parkingv1.LotType
	}{
		{domain.LotTypeUnderground, parkingv1.LotType_LOT_TYPE_UNDERGROUND},
		{domain.LotTypeGround, parkingv1.LotType_LOT_TYPE_GROUND},
		{domain.LotTypeStereo, parkingv1.LotType_LOT_TYPE_STEREO},
		{"unknown", parkingv1.LotType_LOT_TYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, domainLotTypeToProto(tt.input), "input: %s", tt.input)
	}
}

func TestDomainStatusToProto_AllCases(t *testing.T) {
	tests := []struct {
		input    domain.ParkingLotStatus
		expected parkingv1.ParkingLotStatus
	}{
		{domain.ParkingLotStatusActive, parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_ACTIVE},
		{domain.ParkingLotStatusInactive, parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_INACTIVE},
		{"unknown", parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, domainStatusToProto(tt.input), "input: %s", tt.input)
	}
}

func TestDomainLotTypeFromProto_Unspecified(t *testing.T) {
	assert.Equal(t, domain.LotType(""), domainLotTypeFromProto(parkingv1.LotType_LOT_TYPE_UNSPECIFIED))
}

func TestDomainStatusFromProto_Unspecified(t *testing.T) {
	assert.Equal(t, domain.ParkingLotStatus(""), domainStatusFromProto(parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_UNSPECIFIED))
}

func strPtr(s string) *string {
	return &s
}
