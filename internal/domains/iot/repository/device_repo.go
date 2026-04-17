package repository

import (
	"context"
	"time"

	"github.com/parkhub/api/internal/domains/iot/domain"
	"github.com/parkhub/api/internal/domains/iot/repository/dao"
)

type deviceRepo struct {
	dao dao.DeviceDAO
}

func NewDeviceRepo(d dao.DeviceDAO) DeviceRepo {
	return &deviceRepo{dao: d}
}

func (r *deviceRepo) Create(ctx context.Context, device *domain.Device) error {
	return r.dao.Insert(ctx, toDAODevice(device))
}

func (r *deviceRepo) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	d, err := r.dao.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDomainDevice(d), nil
}

func (r *deviceRepo) List(ctx context.Context, filter DeviceFilter, page, pageSize int) ([]*domain.Device, int64, error) {
	daoFilter := dao.DeviceFilter{
		TenantID:     filter.TenantID,
		Status:       string(filter.Status),
		ParkingLotID: filter.ParkingLotID,
		Keyword:      filter.Keyword,
	}
	records, total, err := r.dao.FindAll(ctx, daoFilter, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	devices := make([]*domain.Device, 0, len(records))
	for _, rec := range records {
		devices = append(devices, toDomainDevice(rec))
	}
	return devices, total, nil
}

func (r *deviceRepo) Update(ctx context.Context, device *domain.Device) error {
	return r.dao.Update(ctx, toDAODevice(device))
}

func (r *deviceRepo) Delete(ctx context.Context, id string) error {
	return r.dao.Delete(ctx, id)
}

func (r *deviceRepo) DeleteBatch(ctx context.Context, ids []string) (int64, error) {
	return r.dao.DeleteBatch(ctx, ids)
}

func (r *deviceRepo) CountByStatus(ctx context.Context, tenantID string) (int64, int64, int64, int64, error) {
	return r.dao.CountByStatus(ctx, tenantID)
}

func (r *deviceRepo) UpdateStatus(ctx context.Context, id, status string) error {
	return r.dao.UpdateStatus(ctx, id, status)
}

func (r *deviceRepo) UpdateStatusBatch(ctx context.Context, ids []string, status string) (int64, error) {
	return r.dao.UpdateStatusBatch(ctx, ids, status)
}

func (r *deviceRepo) UnbindByDeviceIDs(ctx context.Context, ids []string) error {
	return r.dao.UnbindByDeviceIDs(ctx, ids)
}

func toDomainDevice(d *dao.Device) *domain.Device {
	if d == nil {
		return nil
	}
	var lastHeartbeat *time.Time
	if d.LastHeartbeatAt != nil {
		t := time.UnixMilli(*d.LastHeartbeatAt)
		lastHeartbeat = &t
	}
	return &domain.Device{
		ID:              d.ID,
		TenantID:        d.TenantID,
		Name:            d.Name,
		Type:            domain.DeviceType(d.Type),
		Status:          domain.DeviceStatus(d.Status),
		FirmwareVersion: d.FirmwareVersion,
		LastHeartbeat:   lastHeartbeat,
		ParkingLotID:    d.ParkingLotID,
		GateID:          d.GateID,
		CreatedAt:       time.UnixMilli(d.CreatedAt),
		UpdatedAt:       time.UnixMilli(d.UpdatedAt),
	}
}

func toDAODevice(d *domain.Device) *dao.Device {
	if d == nil {
		return nil
	}
	var lastHeartbeatAt *int64
	if d.LastHeartbeat != nil {
		ms := d.LastHeartbeat.UnixMilli()
		lastHeartbeatAt = &ms
	}
	return &dao.Device{
		ID:              d.ID,
		TenantID:        d.TenantID,
		Name:            d.Name,
		Type:            string(d.Type),
		Status:          string(d.Status),
		FirmwareVersion: d.FirmwareVersion,
		LastHeartbeatAt: lastHeartbeatAt,
		ParkingLotID:    d.ParkingLotID,
		GateID:          d.GateID,
		CreatedAt:       d.CreatedAt.UnixMilli(),
		UpdatedAt:       d.UpdatedAt.UnixMilli(),
	}
}
