package repository

import (
	"context"

	"github.com/parkhub/api/services/parking/internal/domain"
	"github.com/parkhub/api/services/parking/internal/repository/dao"
)

type parkingLotRepo struct {
	dao dao.ParkingLotDAO
}

func NewParkingLotRepo(d dao.ParkingLotDAO) ParkingLotRepo {
	return &parkingLotRepo{dao: d}
}

func (r *parkingLotRepo) Create(ctx context.Context, lot *domain.ParkingLot) error {
	return r.dao.Insert(ctx, toEntity(lot))
}

func (r *parkingLotRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.ParkingLot, error) {
	d, err := r.dao.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return toDomain(d), nil
}

func (r *parkingLotRepo) List(ctx context.Context, filter ParkingLotFilter, page, pageSize int) ([]*domain.ParkingLot, int64, error) {
	daoFilter := dao.ParkingLotFilter{
		TenantID: filter.TenantID,
		Status:   string(filter.Status),
		LotType:  string(filter.LotType),
		Keyword:  filter.Keyword,
	}
	records, total, err := r.dao.FindAll(ctx, daoFilter, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	lots := make([]*domain.ParkingLot, 0, len(records))
	for _, rec := range records {
		lots = append(lots, toDomain(rec))
	}
	return lots, total, nil
}

func (r *parkingLotRepo) Update(ctx context.Context, lot *domain.ParkingLot) error {
	return r.dao.Update(ctx, toEntity(lot))
}

func (r *parkingLotRepo) Delete(ctx context.Context, tenantID, id string) error {
	return r.dao.Delete(ctx, tenantID, id)
}

func (r *parkingLotRepo) SumStats(ctx context.Context, tenantID string) (int64, int64, error) {
	return r.dao.SumStats(ctx, tenantID)
}

func toDomain(d *dao.ParkingLot) *domain.ParkingLot {
	if d == nil {
		return nil
	}
	return &domain.ParkingLot{
		ID:              d.ID,
		TenantID:        d.TenantID,
		Name:            d.Name,
		Address:         d.Address,
		TotalSpaces:     d.TotalSpaces,
		AvailableSpaces: d.AvailableSpaces,
		LotType:         domain.LotType(d.LotType),
		Status:          domain.ParkingLotStatus(d.Status),
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

func toEntity(lot *domain.ParkingLot) *dao.ParkingLot {
	if lot == nil {
		return nil
	}
	return &dao.ParkingLot{
		ID:              lot.ID,
		TenantID:        lot.TenantID,
		Name:            lot.Name,
		Address:         lot.Address,
		TotalSpaces:     lot.TotalSpaces,
		AvailableSpaces: lot.AvailableSpaces,
		LotType:         string(lot.LotType),
		Status:          string(lot.Status),
		CreatedAt:       lot.CreatedAt,
		UpdatedAt:       lot.UpdatedAt,
	}
}
