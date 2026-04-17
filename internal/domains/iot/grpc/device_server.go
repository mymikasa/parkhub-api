package grpc

import (
	"context"

	"github.com/parkhub/api/internal/domains/iot/errs"
	"github.com/parkhub/api/internal/domains/iot/service"
	commonv1 "github.com/parkhub/api/internal/gen/api/proto/common/v1"
	iotv1 "github.com/parkhub/api/internal/gen/api/proto/iot/v1"
	"github.com/parkhub/api/pkg/grpcutil"
	"google.golang.org/grpc/codes"
)

type DeviceGRPCServer struct {
	iotv1.UnimplementedDeviceServiceServer
	svc service.DeviceService
}

func NewDeviceGRPCServer(svc service.DeviceService) *DeviceGRPCServer {
	return &DeviceGRPCServer{svc: svc}
}

var deviceErrorMappings = []grpcutil.ErrorMapping{
	{Target: errs.ErrDeviceNotFound, Code: codes.NotFound},
	{Target: errs.ErrDeviceIDDuplicate, Code: codes.AlreadyExists},
	{Target: errs.ErrDeviceNotBound, Code: codes.FailedPrecondition},
	{Target: errs.ErrDeviceMustUnbind, Code: codes.FailedPrecondition},
	{Target: errs.ErrDeviceOffline, Code: codes.FailedPrecondition},
	{Target: errs.ErrInvalidCommand, Code: codes.InvalidArgument},
}

func toDeviceGRPCError(err error) error {
	return grpcutil.ToGRPCError(err, deviceErrorMappings)
}

func (s *DeviceGRPCServer) CreateDevice(ctx context.Context, req *iotv1.CreateDeviceRequest) (*iotv1.CreateDeviceResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetId() == "" {
		return nil, grpcutil.ToGRPCError(errs.ErrDeviceNotFound, nil)
	}

	d, err := s.svc.Create(ctx, &service.CreateDeviceRequest{
		TenantID:        tenantID,
		ID:              req.GetId(),
		Name:            req.GetName(),
		Type:            deviceTypeFromProto(req.GetType()),
		FirmwareVersion: req.GetFirmwareVersion(),
	})
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.CreateDeviceResponse{Device: toProtoDevice(d)}, nil
}

func (s *DeviceGRPCServer) GetDevice(ctx context.Context, req *iotv1.GetDeviceRequest) (*iotv1.GetDeviceResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	d, err := s.svc.GetByID(ctx, &service.GetDeviceRequest{TenantID: tenantID, ID: req.GetId()})
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.GetDeviceResponse{Device: toProtoDevice(d)}, nil
}

func (s *DeviceGRPCServer) ListDevices(ctx context.Context, req *iotv1.ListDevicesRequest) (*iotv1.ListDevicesResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	page := int32(1)
	pageSize := int32(20)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PageSize > 0 {
			pageSize = req.Pagination.PageSize
		}
	}

	resp, err := s.svc.List(ctx, &service.ListDevicesRequest{
		TenantID:     tenantID,
		Status:       deviceStatusFromProto(req.GetStatus()),
		ParkingLotID: req.GetParkingLotId(),
		Keyword:      req.GetKeyword(),
		Page:         int(page),
		PageSize:     int(pageSize),
	})
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}

	devices := make([]*iotv1.Device, 0, len(resp.Devices))
	for _, d := range resp.Devices {
		devices = append(devices, toProtoDevice(d))
	}

	return &iotv1.ListDevicesResponse{
		Devices: devices,
		Pagination: &commonv1.PaginationResponse{
			Page:       int32(resp.Page),
			PageSize:   int32(resp.PageSize),
			Total:      resp.Total,
			TotalPages: int32(resp.TotalPages),
		},
	}, nil
}

func (s *DeviceGRPCServer) UpdateDeviceName(ctx context.Context, req *iotv1.UpdateDeviceNameRequest) (*iotv1.UpdateDeviceNameResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	d, err := s.svc.UpdateName(ctx, &service.UpdateDeviceNameRequest{TenantID: tenantID, ID: req.GetId(), Name: req.GetName()})
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.UpdateDeviceNameResponse{Device: toProtoDevice(d)}, nil
}

func (s *DeviceGRPCServer) BindDevice(ctx context.Context, req *iotv1.BindDeviceRequest) (*iotv1.BindDeviceResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	d, err := s.svc.Bind(ctx, &service.BindDeviceRequest{
		TenantID:     tenantID,
		ID:           req.GetId(),
		ParkingLotID: req.GetParkingLotId(),
		GateID:       req.GetGateId(),
	})
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.BindDeviceResponse{Device: toProtoDevice(d)}, nil
}

func (s *DeviceGRPCServer) UnbindDevice(ctx context.Context, req *iotv1.UnbindDeviceRequest) (*iotv1.UnbindDeviceResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	d, err := s.svc.Unbind(ctx, &service.UnbindDeviceRequest{TenantID: tenantID, ID: req.GetId()})
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.UnbindDeviceResponse{Device: toProtoDevice(d)}, nil
}

func (s *DeviceGRPCServer) DisableDevice(ctx context.Context, req *iotv1.DisableDeviceRequest) (*iotv1.DisableDeviceResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	d, err := s.svc.Disable(ctx, &service.ChangeDeviceStatusRequest{TenantID: tenantID, ID: req.GetId()})
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.DisableDeviceResponse{Device: toProtoDevice(d)}, nil
}

func (s *DeviceGRPCServer) EnableDevice(ctx context.Context, req *iotv1.EnableDeviceRequest) (*iotv1.EnableDeviceResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	d, err := s.svc.Enable(ctx, &service.ChangeDeviceStatusRequest{TenantID: tenantID, ID: req.GetId()})
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.EnableDeviceResponse{Device: toProtoDevice(d)}, nil
}

func (s *DeviceGRPCServer) DeleteDevice(ctx context.Context, req *iotv1.DeleteDeviceRequest) (*iotv1.DeleteDeviceResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.svc.Delete(ctx, &service.DeleteDeviceRequest{TenantID: tenantID, ID: req.GetId()}); err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.DeleteDeviceResponse{}, nil
}

func (s *DeviceGRPCServer) BatchDisableDevices(ctx context.Context, req *iotv1.BatchChangeDeviceStatusRequest) (*iotv1.BatchChangeDeviceStatusResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.svc.BatchDisable(ctx, &service.BatchChangeDeviceStatusRequest{TenantID: tenantID, IDs: req.GetIds()}); err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.BatchChangeDeviceStatusResponse{Affected: int32(len(req.GetIds()))}, nil
}

func (s *DeviceGRPCServer) BatchEnableDevices(ctx context.Context, req *iotv1.BatchChangeDeviceStatusRequest) (*iotv1.BatchChangeDeviceStatusResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.svc.BatchEnable(ctx, &service.BatchChangeDeviceStatusRequest{TenantID: tenantID, IDs: req.GetIds()}); err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.BatchChangeDeviceStatusResponse{Affected: int32(len(req.GetIds()))}, nil
}

func (s *DeviceGRPCServer) BatchDeleteDevices(ctx context.Context, req *iotv1.BatchDeleteDeviceRequest) (*iotv1.BatchDeleteDeviceResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.svc.BatchDelete(ctx, &service.BatchDeleteDeviceRequest{TenantID: tenantID, IDs: req.GetIds()}); err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.BatchDeleteDeviceResponse{Affected: int32(len(req.GetIds()))}, nil
}

func (s *DeviceGRPCServer) BatchBindDevices(ctx context.Context, req *iotv1.BatchBindRequest) (*iotv1.BatchBindResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	bindings := make([]service.Binding, 0, len(req.GetBindings()))
	for _, b := range req.GetBindings() {
		bindings = append(bindings, service.Binding{
			ID:           b.GetId(),
			ParkingLotID: b.GetParkingLotId(),
			GateID:       b.GetGateId(),
		})
	}

	if err := s.svc.BatchBind(ctx, &service.BatchBindDeviceRequest{TenantID: tenantID, Bindings: bindings}); err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.BatchBindResponse{Affected: int32(len(bindings))}, nil
}

func (s *DeviceGRPCServer) GetDeviceStats(ctx context.Context, req *iotv1.GetDeviceStatsRequest) (*iotv1.GetDeviceStatsResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	stats, err := s.svc.GetStats(ctx, tenantID)
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.GetDeviceStatsResponse{
		Total:    stats.Total,
		Active:   stats.Active,
		Offline:  stats.Offline,
		Pending:  stats.Pending,
		Disabled: stats.Disabled,
	}, nil
}

func (s *DeviceGRPCServer) SendDeviceCommand(ctx context.Context, req *iotv1.SendDeviceCommandRequest) (*iotv1.SendDeviceCommandResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.svc.SendCommand(ctx, tenantID, req.GetId(), req.GetAction())
	if err != nil {
		return nil, toDeviceGRPCError(err)
	}
	return &iotv1.SendDeviceCommandResponse{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}
