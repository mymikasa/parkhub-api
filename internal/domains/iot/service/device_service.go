package service

import (
	"context"
	"math"

	"github.com/parkhub/api/internal/domains/iot/domain"
	"github.com/parkhub/api/internal/domains/iot/errs"
	"github.com/parkhub/api/internal/domains/iot/repository"
)

type deviceService struct {
	repo repository.DeviceRepo
}

func NewDeviceService(repo repository.DeviceRepo) DeviceService {
	return &deviceService{repo: repo}
}

func (s *deviceService) Create(ctx context.Context, req *CreateDeviceRequest) (*domain.Device, error) {
	d := domain.NewDevice(req.ID, req.Name, req.Type, req.FirmwareVersion)
	d.TenantID = req.TenantID
	if err := s.repo.Create(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *deviceService) GetByID(ctx context.Context, req *GetDeviceRequest) (*domain.Device, error) {
	return s.repo.GetByID(ctx, req.TenantID, req.ID)
}

func (s *deviceService) List(ctx context.Context, req *ListDevicesRequest) (*DeviceListResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filter := repository.DeviceFilter{
		TenantID:     req.TenantID,
		Status:       req.Status,
		ParkingLotID: req.ParkingLotID,
		Keyword:      req.Keyword,
	}

	devices, total, err := s.repo.List(ctx, filter, page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &DeviceListResponse{
		Devices:    devices,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *deviceService) UpdateName(ctx context.Context, req *UpdateDeviceNameRequest) (*domain.Device, error) {
	d, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		return nil, err
	}
	d.UpdateName(req.Name)
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *deviceService) Bind(ctx context.Context, req *BindDeviceRequest) (*domain.Device, error) {
	d, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		return nil, err
	}
	d.TenantID = req.TenantID
	if err := d.Bind(req.ParkingLotID, req.GateID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *deviceService) Unbind(ctx context.Context, req *UnbindDeviceRequest) (*domain.Device, error) {
	d, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if err := d.Unbind(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *deviceService) Disable(ctx context.Context, req *ChangeDeviceStatusRequest) (*domain.Device, error) {
	d, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if err := d.Disable(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *deviceService) Enable(ctx context.Context, req *ChangeDeviceStatusRequest) (*domain.Device, error) {
	d, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if err := d.Enable(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *deviceService) Delete(ctx context.Context, req *DeleteDeviceRequest) error {
	d, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		return err
	}
	if d.IsBound() {
		return errs.ErrDeviceMustUnbind
	}
	return s.repo.Delete(ctx, req.TenantID, req.ID)
}

func (s *deviceService) BatchDisable(ctx context.Context, req *BatchChangeDeviceStatusRequest) error {
	_, err := s.repo.UpdateStatusBatch(ctx, req.TenantID, req.IDs, string(domain.DeviceStatusDisabled))
	return err
}

func (s *deviceService) BatchEnable(ctx context.Context, req *BatchChangeDeviceStatusRequest) error {
	_, err := s.repo.UpdateStatusBatch(ctx, req.TenantID, req.IDs, string(domain.DeviceStatusActive))
	return err
}

func (s *deviceService) BatchDelete(ctx context.Context, req *BatchDeleteDeviceRequest) error {
	if err := s.repo.UnbindByDeviceIDs(ctx, req.TenantID, req.IDs); err != nil {
		return err
	}
	_, err := s.repo.DeleteBatch(ctx, req.TenantID, req.IDs)
	return err
}

func (s *deviceService) BatchBind(ctx context.Context, req *BatchBindDeviceRequest) error {
	for _, b := range req.Bindings {
		d, err := s.repo.GetByID(ctx, req.TenantID, b.ID)
		if err != nil {
			return err
		}
		d.TenantID = req.TenantID
		if err := d.Bind(b.ParkingLotID, b.GateID); err != nil {
			return err
		}
		if err := s.repo.Update(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

func (s *deviceService) GetStats(ctx context.Context, tenantID string) (*DeviceStatsResponse, error) {
	pending, active, offline, disabled, err := s.repo.CountByStatus(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &DeviceStatsResponse{
		Total:    pending + active + offline + disabled,
		Active:   active,
		Offline:  offline,
		Pending:  pending,
		Disabled: disabled,
	}, nil
}

func (s *deviceService) SendCommand(ctx context.Context, tenantID, deviceID, action string) (*CommandResponse, error) {
	if action != "up" && action != "down" {
		return nil, errs.ErrInvalidCommand
	}
	d, err := s.repo.GetByID(ctx, tenantID, deviceID)
	if err != nil {
		return nil, err
	}
	if d.Status == domain.DeviceStatusOffline || d.Status == domain.DeviceStatusDisabled {
		return nil, errs.ErrDeviceOffline
	}
	msg := "抬杆指令已发送"
	if action == "down" {
		msg = "落杆指令已发送"
	}
	return &CommandResponse{Success: true, Message: msg}, nil
}
