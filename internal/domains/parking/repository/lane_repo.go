package repository

import (
	"context"

	"github.com/parkhub/api/internal/domains/parking/domain"
	"github.com/parkhub/api/internal/domains/parking/repository/dao"
)

type laneRepo struct {
	dao dao.LaneDAO
}

func NewLaneRepo(d dao.LaneDAO) LaneRepo {
	return &laneRepo{dao: d}
}

func (r *laneRepo) Create(ctx context.Context, lane *domain.Lane) error {
	return r.dao.Insert(ctx, toLaneEntity(lane))
}

func (r *laneRepo) GetByID(ctx context.Context, tenantID, id string) (*domain.Lane, error) {
	d, err := r.dao.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return toLaneDomain(d), nil
}

func (r *laneRepo) ListByParkingLotID(ctx context.Context, tenantID, parkingLotID string) ([]*domain.Lane, error) {
	records, err := r.dao.FindByParkingLotID(ctx, tenantID, parkingLotID)
	if err != nil {
		return nil, err
	}
	lanes := make([]*domain.Lane, 0, len(records))
	for _, rec := range records {
		lanes = append(lanes, toLaneDomain(rec))
	}
	return lanes, nil
}

func (r *laneRepo) Update(ctx context.Context, lane *domain.Lane) error {
	return r.dao.Update(ctx, toLaneEntity(lane))
}

func (r *laneRepo) Delete(ctx context.Context, tenantID, id string) error {
	return r.dao.Delete(ctx, tenantID, id)
}

func (r *laneRepo) ExistsByName(ctx context.Context, parkingLotID, name string) (bool, error) {
	return r.dao.ExistsByName(ctx, parkingLotID, name)
}

func (r *laneRepo) CountByParkingLots(ctx context.Context, tenantID string, parkingLotIDs []string) (map[string]*LaneCountResult, error) {
	daoCounts, err := r.dao.CountByParkingLots(ctx, tenantID, parkingLotIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*LaneCountResult, len(daoCounts))
	for k, v := range daoCounts {
		result[k] = &LaneCountResult{EntryCount: v.Entry, ExitCount: v.Exit}
	}
	return result, nil
}

func toLaneDomain(d *dao.Lane) *domain.Lane {
	if d == nil {
		return nil
	}
	return &domain.Lane{
		ID:           d.ID,
		TenantID:     d.TenantID,
		ParkingLotID: d.ParkingLotID,
		Name:         d.Name,
		Type:         domain.LaneType(d.LaneType),
		DeviceID:     d.DeviceID,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

func toLaneEntity(l *domain.Lane) *dao.Lane {
	if l == nil {
		return nil
	}
	return &dao.Lane{
		ID:           l.ID,
		TenantID:     l.TenantID,
		ParkingLotID: l.ParkingLotID,
		Name:         l.Name,
		LaneType:     string(l.Type),
		DeviceID:     l.DeviceID,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}
