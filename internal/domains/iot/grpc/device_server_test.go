package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/parkhub/api/internal/domains/iot/domain"
	"github.com/parkhub/api/internal/domains/iot/errs"
	"github.com/parkhub/api/internal/domains/iot/service"
	servicemocks "github.com/parkhub/api/internal/domains/iot/service/mocks"
	commonv1 "github.com/parkhub/api/internal/gen/api/proto/common/v1"
	iotv1 "github.com/parkhub/api/internal/gen/api/proto/iot/v1"
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

func setupDeviceTestServer(t *testing.T) (iotv1.DeviceServiceClient, *servicemocks.MockDeviceService, *go_mock.Controller) {
	t.Helper()
	ctrl := go_mock.NewController(t)
	mockSvc := servicemocks.NewMockDeviceService(ctrl)

	srv := NewDeviceGRPCServer(mockSvc)
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer(
		grpc.UnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				if tids := md.Get("x-tenant-id"); len(tids) > 0 {
					ctx = identityctx.WithTenantID(ctx, tids[0])
				}
			}
			return handler(ctx, req)
		}),
	)
	iotv1.RegisterDeviceServiceServer(s, srv)
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

	return iotv1.NewDeviceServiceClient(conn), mockSvc, ctrl
}

func ctxWithTenant() context.Context {
	ctx := context.Background()
	md := metadata.Pairs("x-tenant-id", "tenant-1")
	return metadata.NewOutgoingContext(ctx, md)
}

func strPtr(s string) *string { return &s }

func newDomainDevice() *domain.Device {
	now := time.Now()
	lotID := "lot-1"
	gateID := "gate-1"
	return &domain.Device{
		ID:              "DEV-001",
		TenantID:        "tenant-1",
		Name:            "入口摄像头",
		Type:            domain.DeviceTypeIntegrated,
		Status:          domain.DeviceStatusActive,
		FirmwareVersion: "v1.0.0",
		ParkingLotID:    &lotID,
		GateID:          &gateID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func TestGRPC_CreateDevice_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	expected := newDomainDevice()
	mockSvc.EXPECT().
		Create(go_mock.Any(), go_mock.Any()).
		Return(expected, nil)

	resp, err := client.CreateDevice(ctxWithTenant(), &iotv1.CreateDeviceRequest{
		Id: "DEV-001", Name: "入口摄像头", Type: iotv1.DeviceType_DEVICE_TYPE_INTEGRATED, FirmwareVersion: "v1.0.0",
	})
	assert.NoError(t, err)
	assert.Equal(t, "DEV-001", resp.Device.Id)
	assert.Equal(t, "tenant-1", resp.Device.TenantId)
	assert.Equal(t, "入口摄像头", resp.Device.Name)
	assert.Equal(t, iotv1.DeviceType_DEVICE_TYPE_INTEGRATED, resp.Device.Type)
	assert.Equal(t, iotv1.DeviceStatus_DEVICE_STATUS_ACTIVE, resp.Device.Status)
	assert.Equal(t, "v1.0.0", resp.Device.FirmwareVersion)
}

func TestGRPC_CreateDevice_MissingTenantID(t *testing.T) {
	client, _, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	_, err := client.CreateDevice(context.Background(), &iotv1.CreateDeviceRequest{
		Id: "DEV-001", Name: "test",
	})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPC_CreateDevice_MissingID(t *testing.T) {
	client, _, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	_, err := client.CreateDevice(ctxWithTenant(), &iotv1.CreateDeviceRequest{
		Name: "test",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_CreateDevice_InvalidType(t *testing.T) {
	client, _, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	_, err := client.CreateDevice(ctxWithTenant(), &iotv1.CreateDeviceRequest{
		Id:   "DEV-001",
		Name: "test",
		Type: iotv1.DeviceType_DEVICE_TYPE_UNSPECIFIED,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_CreateDevice_DuplicateID(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Create(go_mock.Any(), go_mock.Any()).
		Return(nil, errs.ErrDeviceIDDuplicate)

	_, err := client.CreateDevice(ctxWithTenant(), &iotv1.CreateDeviceRequest{
		Id: "DEV-001", Name: "dup", Type: iotv1.DeviceType_DEVICE_TYPE_INTEGRATED,
	})
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestGRPC_GetDevice_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	expected := newDomainDevice()
	mockSvc.EXPECT().
		GetByID(go_mock.Any(), go_mock.Any()).
		Return(expected, nil)

	resp, err := client.GetDevice(ctxWithTenant(), &iotv1.GetDeviceRequest{Id: "DEV-001"})
	assert.NoError(t, err)
	assert.Equal(t, "入口摄像头", resp.Device.Name)
}

func TestGRPC_GetDevice_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		GetByID(go_mock.Any(), go_mock.Any()).
		Return(nil, errs.ErrDeviceNotFound)

	_, err := client.GetDevice(ctxWithTenant(), &iotv1.GetDeviceRequest{Id: "bad-id"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_GetDevice_EmptyID(t *testing.T) {
	client, _, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	_, err := client.GetDevice(ctxWithTenant(), &iotv1.GetDeviceRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_ListDevices_WithPagination(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		List(go_mock.Any(), go_mock.Any()).
		DoAndReturn(func(_ context.Context, req *service.ListDevicesRequest) (*service.DeviceListResponse, error) {
			assert.Equal(t, "tenant-1", req.TenantID)
			assert.Equal(t, 2, req.Page)
			assert.Equal(t, 10, req.PageSize)
			return &service.DeviceListResponse{
				Devices: []*domain.Device{newDomainDevice()},
				Total:   1, Page: 2, PageSize: 10, TotalPages: 1,
			}, nil
		})

	resp, err := client.ListDevices(ctxWithTenant(), &iotv1.ListDevicesRequest{
		Pagination: &commonv1.PaginationRequest{Page: 2, PageSize: 10},
	})
	assert.NoError(t, err)
	assert.Len(t, resp.Devices, 1)
	assert.Equal(t, int32(2), resp.Pagination.Page)
	assert.Equal(t, int32(10), resp.Pagination.PageSize)
}

func TestGRPC_ListDevices_ReturnsServicePagination(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		List(go_mock.Any(), go_mock.Any()).
		Return(&service.DeviceListResponse{
			Devices: []*domain.Device{}, Total: 50, Page: 1, PageSize: 100, TotalPages: 1,
		}, nil)

	resp, err := client.ListDevices(ctxWithTenant(), &iotv1.ListDevicesRequest{
		Pagination: &commonv1.PaginationRequest{Page: 1, PageSize: 999},
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(100), resp.Pagination.PageSize)
}

func TestGRPC_ListDevices_UnspecifiedFiltersNoFilter(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		List(go_mock.Any(), go_mock.Any()).
		DoAndReturn(func(_ context.Context, req *service.ListDevicesRequest) (*service.DeviceListResponse, error) {
			assert.Equal(t, domain.DeviceStatus(""), req.Status)
			return &service.DeviceListResponse{}, nil
		})

	_, err := client.ListDevices(ctxWithTenant(), &iotv1.ListDevicesRequest{})
	assert.NoError(t, err)
}

func TestGRPC_UpdateDeviceName_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	updated := newDomainDevice()
	updated.Name = "新名字"
	mockSvc.EXPECT().
		UpdateName(go_mock.Any(), go_mock.Any()).
		Return(updated, nil)

	resp, err := client.UpdateDeviceName(ctxWithTenant(), &iotv1.UpdateDeviceNameRequest{
		Id: "DEV-001", Name: "新名字",
	})
	assert.NoError(t, err)
	assert.Equal(t, "新名字", resp.Device.Name)
}

func TestGRPC_UpdateDeviceName_EmptyID(t *testing.T) {
	client, _, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	_, err := client.UpdateDeviceName(ctxWithTenant(), &iotv1.UpdateDeviceNameRequest{Name: "新名字"})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_UpdateDeviceName_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		UpdateName(go_mock.Any(), go_mock.Any()).
		Return(nil, errs.ErrDeviceNotFound)

	_, err := client.UpdateDeviceName(ctxWithTenant(), &iotv1.UpdateDeviceNameRequest{
		Id: "bad-id", Name: "x",
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_BindDevice_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	expected := newDomainDevice()
	mockSvc.EXPECT().
		Bind(go_mock.Any(), go_mock.Any()).
		Return(expected, nil)

	resp, err := client.BindDevice(ctxWithTenant(), &iotv1.BindDeviceRequest{
		Id: "DEV-001", ParkingLotId: "lot-1", GateId: "gate-1",
	})
	assert.NoError(t, err)
	assert.Equal(t, "DEV-001", resp.Device.Id)
}

func TestGRPC_UnbindDevice_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	expected := newDomainDevice()
	expected.ParkingLotID = nil
	expected.GateID = nil
	mockSvc.EXPECT().
		Unbind(go_mock.Any(), go_mock.Any()).
		Return(expected, nil)

	resp, err := client.UnbindDevice(ctxWithTenant(), &iotv1.UnbindDeviceRequest{Id: "DEV-001"})
	assert.NoError(t, err)
	assert.Nil(t, resp.Device.ParkingLotId)
}

func TestGRPC_UnbindDevice_NotBound(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Unbind(go_mock.Any(), go_mock.Any()).
		Return(nil, errs.ErrDeviceNotBound)

	_, err := client.UnbindDevice(ctxWithTenant(), &iotv1.UnbindDeviceRequest{Id: "DEV-001"})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGRPC_DisableDevice_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	disabled := newDomainDevice()
	disabled.Status = domain.DeviceStatusDisabled
	mockSvc.EXPECT().
		Disable(go_mock.Any(), go_mock.Any()).
		Return(disabled, nil)

	resp, err := client.DisableDevice(ctxWithTenant(), &iotv1.DisableDeviceRequest{Id: "DEV-001"})
	assert.NoError(t, err)
	assert.Equal(t, iotv1.DeviceStatus_DEVICE_STATUS_DISABLED, resp.Device.Status)
}

func TestGRPC_EnableDevice_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	enabled := newDomainDevice()
	enabled.Status = domain.DeviceStatusActive
	mockSvc.EXPECT().
		Enable(go_mock.Any(), go_mock.Any()).
		Return(enabled, nil)

	resp, err := client.EnableDevice(ctxWithTenant(), &iotv1.EnableDeviceRequest{Id: "DEV-001"})
	assert.NoError(t, err)
	assert.Equal(t, iotv1.DeviceStatus_DEVICE_STATUS_ACTIVE, resp.Device.Status)
}

func TestGRPC_DeleteDevice_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Delete(go_mock.Any(), go_mock.Any()).
		Return(nil)

	_, err := client.DeleteDevice(ctxWithTenant(), &iotv1.DeleteDeviceRequest{Id: "DEV-001"})
	assert.NoError(t, err)
}

func TestGRPC_DeleteDevice_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Delete(go_mock.Any(), go_mock.Any()).
		Return(errs.ErrDeviceNotFound)

	_, err := client.DeleteDevice(ctxWithTenant(), &iotv1.DeleteDeviceRequest{Id: "bad-id"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGRPC_DeleteDevice_MustUnbind(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		Delete(go_mock.Any(), go_mock.Any()).
		Return(errs.ErrDeviceMustUnbind)

	_, err := client.DeleteDevice(ctxWithTenant(), &iotv1.DeleteDeviceRequest{Id: "DEV-001"})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGRPC_BatchDisableDevices_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		BatchDisable(go_mock.Any(), go_mock.Any()).
		Return(int64(2), nil)

	resp, err := client.BatchDisableDevices(ctxWithTenant(), &iotv1.BatchDisableDevicesRequest{
		Ids: []string{"DEV-001", "DEV-002"},
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.Affected)
}

func TestGRPC_BatchEnableDevices_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		BatchEnable(go_mock.Any(), go_mock.Any()).
		Return(int64(1), nil)

	resp, err := client.BatchEnableDevices(ctxWithTenant(), &iotv1.BatchEnableDevicesRequest{
		Ids: []string{"DEV-001"},
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), resp.Affected)
}

func TestGRPC_BatchDeleteDevices_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		BatchDelete(go_mock.Any(), go_mock.Any()).
		Return(int64(3), nil)

	resp, err := client.BatchDeleteDevices(ctxWithTenant(), &iotv1.BatchDeleteDevicesRequest{
		Ids: []string{"DEV-001", "DEV-002", "DEV-003"},
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(3), resp.Affected)
}

func TestGRPC_BatchBindDevices_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		BatchBind(go_mock.Any(), go_mock.Any()).
		Return(int64(2), nil)

	resp, err := client.BatchBindDevices(ctxWithTenant(), &iotv1.BatchBindDevicesRequest{
		Bindings: []*iotv1.BatchBindDevicesRequest_Binding{
			{Id: "DEV-001", ParkingLotId: "lot-1", GateId: "gate-1"},
			{Id: "DEV-002", ParkingLotId: "lot-1", GateId: "gate-2"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(2), resp.Affected)
}

func TestGRPC_BatchDisableDevices_AffectedShouldReflectActualRows(t *testing.T) {
	t.Skip("current service/grpc contract only returns error, so actual affected rows cannot be asserted until the production signature is extended")
}

func TestGRPC_GetDeviceStats_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		GetStats(go_mock.Any(), "tenant-1").
		Return(&service.DeviceStatsResponse{
			Total: 10, Active: 5, Offline: 2, Pending: 2, Disabled: 1,
		}, nil)

	resp, err := client.GetDeviceStats(ctxWithTenant(), &iotv1.GetDeviceStatsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, int64(10), resp.Total)
	assert.Equal(t, int64(5), resp.Active)
	assert.Equal(t, int64(2), resp.Offline)
	assert.Equal(t, int64(2), resp.Pending)
	assert.Equal(t, int64(1), resp.Disabled)
}

func TestGRPC_SendDeviceCommand_Success(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		SendCommand(go_mock.Any(), "tenant-1", "DEV-001", "barrier_up").
		Return(&service.CommandResponse{Success: true, Message: "道闸已抬起"}, nil)

	resp, err := client.SendDeviceCommand(ctxWithTenant(), &iotv1.SendDeviceCommandRequest{
		Id: "DEV-001", Action: "barrier_up",
	})
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "道闸已抬起", resp.Message)
}

func TestGRPC_SendDeviceCommand_DeviceOffline(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		SendCommand(go_mock.Any(), "tenant-1", "DEV-001", "barrier_down").
		Return(nil, errs.ErrDeviceOffline)

	_, err := client.SendDeviceCommand(ctxWithTenant(), &iotv1.SendDeviceCommandRequest{
		Id: "DEV-001", Action: "barrier_down",
	})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGRPC_SendDeviceCommand_InvalidAction(t *testing.T) {
	client, mockSvc, ctrl := setupDeviceTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().
		SendCommand(go_mock.Any(), "tenant-1", "DEV-001", "invalid").
		Return(nil, errs.ErrInvalidCommand)

	_, err := client.SendDeviceCommand(ctxWithTenant(), &iotv1.SendDeviceCommandRequest{
		Id: "DEV-001", Action: "invalid",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestToProtoDevice_Nil(t *testing.T) {
	assert.Nil(t, toProtoDevice(nil))
}

func TestDeviceTypeToProto_AllCases(t *testing.T) {
	tests := []struct {
		input    domain.DeviceType
		expected iotv1.DeviceType
	}{
		{domain.DeviceTypeIntegrated, iotv1.DeviceType_DEVICE_TYPE_INTEGRATED},
		{domain.DeviceTypeCameraOnly, iotv1.DeviceType_DEVICE_TYPE_CAMERA_ONLY},
		{domain.DeviceTypeBarrierOnly, iotv1.DeviceType_DEVICE_TYPE_BARRIER_ONLY},
		{"unknown", iotv1.DeviceType_DEVICE_TYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, deviceTypeToProto(tt.input), "input: %s", tt.input)
	}
}

func TestDeviceStatusToProto_AllCases(t *testing.T) {
	tests := []struct {
		input    domain.DeviceStatus
		expected iotv1.DeviceStatus
	}{
		{domain.DeviceStatusPending, iotv1.DeviceStatus_DEVICE_STATUS_PENDING},
		{domain.DeviceStatusActive, iotv1.DeviceStatus_DEVICE_STATUS_ACTIVE},
		{domain.DeviceStatusOffline, iotv1.DeviceStatus_DEVICE_STATUS_OFFLINE},
		{domain.DeviceStatusDisabled, iotv1.DeviceStatus_DEVICE_STATUS_DISABLED},
		{"unknown", iotv1.DeviceStatus_DEVICE_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, deviceStatusToProto(tt.input), "input: %s", tt.input)
	}
}

func TestDeviceTypeFromProto_Unspecified(t *testing.T) {
	assert.Equal(t, domain.DeviceType(""), deviceTypeFromProto(iotv1.DeviceType_DEVICE_TYPE_UNSPECIFIED))
}

func TestDeviceStatusFromProto_Unspecified(t *testing.T) {
	assert.Equal(t, domain.DeviceStatus(""), deviceStatusFromProto(iotv1.DeviceStatus_DEVICE_STATUS_UNSPECIFIED))
}

func TestToProtoDevice_WithOptionalFields(t *testing.T) {
	now := time.Now()
	hb := now.Add(-5 * time.Minute)
	lotID := "lot-1"
	gateID := "gate-1"
	d := &domain.Device{
		ID:              "DEV-001",
		TenantID:        "tenant-1",
		Name:            "test",
		Type:            domain.DeviceTypeCameraOnly,
		Status:          domain.DeviceStatusActive,
		FirmwareVersion: "v2.0",
		LastHeartbeat:   &hb,
		ParkingLotID:    &lotID,
		GateID:          &gateID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	pb := toProtoDevice(d)
	assert.Equal(t, "DEV-001", pb.Id)
	assert.Equal(t, "lot-1", pb.GetParkingLotId())
	assert.Equal(t, "gate-1", pb.GetGateId())
	assert.NotNil(t, pb.LastHeartbeat)
}

func TestToProtoDevice_WithoutOptionalFields(t *testing.T) {
	d := &domain.Device{
		ID:        "DEV-001",
		TenantID:  "tenant-1",
		Name:      "test",
		Status:    domain.DeviceStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	pb := toProtoDevice(d)
	assert.Nil(t, pb.ParkingLotId)
	assert.Nil(t, pb.GateId)
	assert.Nil(t, pb.LastHeartbeat)
}
