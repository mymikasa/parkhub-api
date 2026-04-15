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
	repo repository.ParkingLotRepo
}

func NewParkingLotService(repo repository.ParkingLotRepo) ParkingLotService {
	return &parkingLotService{repo: repo}
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
	return s.repo.GetByID(ctx, tenantID, id)
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

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &ParkingLotListResponse{
		ParkingLots: lots,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
	}, nil
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

	return &ParkingLotStatsResponse{
		TotalSpaces:      totalSpaces,
		AvailableSpaces:  availableSpaces,
		OccupiedVehicles: totalSpaces - availableSpaces,
		TotalGates:       0,
	}, nil
}
