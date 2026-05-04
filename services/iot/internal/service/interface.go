package service

import (
	"context"

	"github.com/parkhub/api/services/iot/internal/domain"
)

type DeviceService interface {
	Create(ctx context.Context, req *CreateDeviceRequest) (*domain.Device, error)
	GetByID(ctx context.Context, req *GetDeviceRequest) (*domain.Device, error)
	List(ctx context.Context, req *ListDevicesRequest) (*DeviceListResponse, error)
	UpdateName(ctx context.Context, req *UpdateDeviceNameRequest) (*domain.Device, error)
	Bind(ctx context.Context, req *BindDeviceRequest) (*domain.Device, error)
	Unbind(ctx context.Context, req *UnbindDeviceRequest) (*domain.Device, error)
	Disable(ctx context.Context, req *ChangeDeviceStatusRequest) (*domain.Device, error)
	Enable(ctx context.Context, req *ChangeDeviceStatusRequest) (*domain.Device, error)
	Delete(ctx context.Context, req *DeleteDeviceRequest) error
	BatchDisable(ctx context.Context, req *BatchChangeDeviceStatusRequest) (int64, error)
	BatchEnable(ctx context.Context, req *BatchChangeDeviceStatusRequest) (int64, error)
	BatchDelete(ctx context.Context, req *BatchDeleteDeviceRequest) (int64, error)
	BatchBind(ctx context.Context, req *BatchBindDeviceRequest) (int64, error)
	GetStats(ctx context.Context, tenantID string) (*DeviceStatsResponse, error)
	SendCommand(ctx context.Context, tenantID, deviceID, action string) (*CommandResponse, error)
}

type CreateDeviceRequest struct {
	TenantID        string
	ID              string
	Name            string
	Type            domain.DeviceType
	FirmwareVersion string
}

type GetDeviceRequest struct {
	TenantID string
	ID       string
}

type ListDevicesRequest struct {
	TenantID     string
	Status       domain.DeviceStatus
	ParkingLotID string
	Keyword      string
	Page         int
	PageSize     int
}

type UpdateDeviceNameRequest struct {
	TenantID string
	ID       string
	Name     string
}

type BindDeviceRequest struct {
	TenantID     string
	ID           string
	ParkingLotID string
	GateID       string
}

type UnbindDeviceRequest struct {
	TenantID string
	ID       string
}

type ChangeDeviceStatusRequest struct {
	TenantID string
	ID       string
}

type DeleteDeviceRequest struct {
	TenantID string
	ID       string
}

type BatchChangeDeviceStatusRequest struct {
	TenantID string
	IDs      []string
}

type BatchDeleteDeviceRequest struct {
	TenantID string
	IDs      []string
}

type BatchBindDeviceRequest struct {
	TenantID string
	Bindings []Binding
}

type Binding struct {
	ID           string
	ParkingLotID string
	GateID       string
}

type DeviceListResponse struct {
	Devices    []*domain.Device
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

type DeviceStatsResponse struct {
	Total    int64
	Active   int64
	Offline  int64
	Pending  int64
	Disabled int64
}

type CommandResponse struct {
	Success bool
	Message string
}
