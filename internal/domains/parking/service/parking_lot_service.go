package service

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/parkhub/api/internal/domains/parking/domain"
	"github.com/parkhub/api/internal/domains/parking/errs"
	"github.com/parkhub/api/internal/domains/parking/repository"
)

type parkingLotService struct {
	repo     repository.ParkingLotRepo
	laneRepo repository.LaneRepo
}

func NewParkingLotService(repo repository.ParkingLotRepo, laneRepo repository.LaneRepo) ParkingLotService {
	return &parkingLotService{repo: repo, laneRepo: laneRepo}
}

func (s *parkingLotService) Create(ctx context.Context, req *CreateParkingLotRequest) (*domain.ParkingLot, error) {
	lot := domain.NewParkingLot(req.Name, req.Address, req.TotalSpaces, req.LotType)
	lot.ID = uuid.New().String()
	lot.TenantID = req.TenantID

	if err := s.repo.Create(ctx, lot); err != nil {
		return nil, err
	}

	return lot, nil
}

func (s *parkingLotService) GetByID(ctx context.Context, tenantID, id string) (*domain.ParkingLot, error) {
	lot, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	s.populateLaneCounts(ctx, tenantID, []*domain.ParkingLot{lot})
	return lot, nil
}

func (s *parkingLotService) List(ctx context.Context, req *ListParkingLotsRequest) (*ParkingLotListResponse, error) {
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

	filter := repository.ParkingLotFilter{
		TenantID: req.TenantID,
		Status:   req.Status,
		LotType:  req.LotType,
		Keyword:  req.Keyword,
	}

	lots, total, err := s.repo.List(ctx, filter, page, pageSize)
	if err != nil {
		return nil, err
	}

	s.populateLaneCounts(ctx, req.TenantID, lots)

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &ParkingLotListResponse{
		ParkingLots: lots,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
	}, nil
}

func (s *parkingLotService) populateLaneCounts(ctx context.Context, tenantID string, lots []*domain.ParkingLot) {
	if s.laneRepo == nil || len(lots) == 0 {
		return
	}
	ids := make([]string, len(lots))
	for i, l := range lots {
		ids[i] = l.ID
	}
	counts, err := s.laneRepo.CountByParkingLots(ctx, tenantID, ids)
	if err != nil {
		return
	}
	for _, l := range lots {
		if c, ok := counts[l.ID]; ok {
			l.EntryCount = c.EntryCount
			l.ExitCount = c.ExitCount
		}
	}
}

func (s *parkingLotService) Update(ctx context.Context, req *UpdateParkingLotRequest) (*domain.ParkingLot, error) {
	lot, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		return nil, err
	}

	if req.Status != nil {
		switch *req.Status {
		case domain.ParkingLotStatusActive:
			if err := lot.Activate(); err != nil {
				return nil, err
			}
		case domain.ParkingLotStatusInactive:
			if err := lot.Deactivate(); err != nil {
				return nil, err
			}
		}
	}

	if req.TotalSpaces != nil {
		occupied := lot.TotalSpaces - lot.AvailableSpaces
		newTotal := *req.TotalSpaces
		if newTotal < occupied {
			return nil, errs.ErrParkingLotInvalidCapacity
		}
		lot.TotalSpaces = newTotal
		lot.AvailableSpaces = newTotal - occupied
	}

	if req.Name != nil {
		lot.Name = *req.Name
	}
	if req.Address != nil {
		lot.Address = *req.Address
	}
	if req.LotType != nil {
		lot.LotType = *req.LotType
	}

	if err := s.repo.Update(ctx, lot); err != nil {
		return nil, err
	}

	return lot, nil
}

func (s *parkingLotService) Delete(ctx context.Context, tenantID, id string) error {
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *parkingLotService) GetStats(ctx context.Context, tenantID string) (*ParkingLotStatsResponse, error) {
	totalSpaces, availableSpaces, err := s.repo.SumStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var totalGates int64
	if s.laneRepo != nil {
		lotIDs, _, err := s.repo.List(ctx, repository.ParkingLotFilter{TenantID: tenantID}, 1, 1000)
		if err == nil && len(lotIDs) > 0 {
			ids := make([]string, len(lotIDs))
			for i, l := range lotIDs {
				ids[i] = l.ID
			}
			counts, err := s.laneRepo.CountByParkingLots(ctx, tenantID, ids)
			if err == nil {
				for _, c := range counts {
					totalGates += int64(c.EntryCount + c.ExitCount)
				}
			}
		}
	}

	return &ParkingLotStatsResponse{
		TotalSpaces:      totalSpaces,
		AvailableSpaces:  availableSpaces,
		OccupiedVehicles: totalSpaces - availableSpaces,
		TotalGates:       totalGates,
	}, nil
}
