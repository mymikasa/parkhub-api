package repository

import (
	"context"

	"github.com/parkhub/api/services/iot/internal/domain"
)

type DeviceRepo interface {
	Transaction(ctx context.Context, fn func(repo DeviceRepo) error) error
	Create(ctx context.Context, device *domain.Device) error
	GetByID(ctx context.Context, tenantID, id string) (*domain.Device, error)
	List(ctx context.Context, filter DeviceFilter, page, pageSize int) ([]*domain.Device, int64, error)
	Update(ctx context.Context, device *domain.Device) error
	Delete(ctx context.Context, tenantID, id string) error
	DeleteBatch(ctx context.Context, tenantID string, ids []string) (int64, error)
	CountByStatus(ctx context.Context, tenantID string) (pending, active, offline, disabled int64, err error)
	UpdateStatus(ctx context.Context, tenantID, id, status string) error
	UpdateStatusBatch(ctx context.Context, tenantID string, ids []string, status string) (int64, error)
	UnbindByDeviceIDs(ctx context.Context, tenantID string, ids []string) error
}

type DeviceFilter struct {
	TenantID     string
	Status       domain.DeviceStatus
	ParkingLotID string
	Keyword      string
}
