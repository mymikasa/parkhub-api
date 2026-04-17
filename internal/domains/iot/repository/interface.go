package repository

import (
	"context"

	"github.com/parkhub/api/internal/domains/iot/domain"
)

//go:generate mockgen -source=./interface.go -package=repomocks -destination=./mocks/device_repo.mock.go DeviceRepo

type DeviceRepo interface {
	Create(ctx context.Context, device *domain.Device) error
	GetByID(ctx context.Context, id string) (*domain.Device, error)
	List(ctx context.Context, filter DeviceFilter, page, pageSize int) ([]*domain.Device, int64, error)
	Update(ctx context.Context, device *domain.Device) error
	Delete(ctx context.Context, id string) error
	DeleteBatch(ctx context.Context, ids []string) (int64, error)
	CountByStatus(ctx context.Context, tenantID string) (pending, active, offline, disabled int64, err error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateStatusBatch(ctx context.Context, ids []string, status string) (int64, error)
	UnbindByDeviceIDs(ctx context.Context, ids []string) error
}

type DeviceFilter struct {
	TenantID     string
	Status       domain.DeviceStatus
	ParkingLotID string
	Keyword      string
}
