package repository

import (
	"context"

	"github.com/parkhub/api/internal/domains/parking/domain"
)

type ParkingLotFilter struct {
	TenantID string
	Status   domain.ParkingLotStatus
	LotType  domain.LotType
	Keyword  string
}

//go:generate mockgen -source=./interface.go -package=repomocks -destination=./mocks/repo.mock.go ParkingLotRepo

type ParkingLotRepo interface {
	Create(ctx context.Context, lot *domain.ParkingLot) error
	GetByID(ctx context.Context, tenantID, id string) (*domain.ParkingLot, error)
	List(ctx context.Context, filter ParkingLotFilter, page, pageSize int) ([]*domain.ParkingLot, int64, error)
	Update(ctx context.Context, lot *domain.ParkingLot) error
	Delete(ctx context.Context, tenantID, id string) error
	SumStats(ctx context.Context, tenantID string) (totalSpaces, availableSpaces int64, err error)
}
