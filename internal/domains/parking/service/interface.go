package service

import (
	"context"

	"github.com/parkhub/api/internal/domains/parking/domain"
)

type CreateParkingLotRequest struct {
	TenantID    string
	Name        string
	Address     string
	TotalSpaces int
	LotType     domain.LotType
}

type UpdateParkingLotRequest struct {
	ID          string
	TenantID    string
	Name        *string
	Address     *string
	TotalSpaces *int
	LotType     *domain.LotType
	Status      *domain.ParkingLotStatus
}

type ListParkingLotsRequest struct {
	TenantID string
	Status   domain.ParkingLotStatus
	LotType  domain.LotType
	Keyword  string
	Page     int
	PageSize int
}

type ParkingLotListResponse struct {
	ParkingLots []*domain.ParkingLot
	Total       int64
	Page        int
	PageSize    int
	TotalPages  int
}

type ParkingLotStatsResponse struct {
	TotalSpaces      int64
	AvailableSpaces  int64
	OccupiedVehicles int64
	TotalGates       int64
}

//go:generate mockgen -source=./interface.go -package=servicemocks -destination=./mocks/parking_lot_service.mock.go ParkingLotService

type ParkingLotService interface {
	Create(ctx context.Context, req *CreateParkingLotRequest) (*domain.ParkingLot, error)
	GetByID(ctx context.Context, tenantID, id string) (*domain.ParkingLot, error)
	List(ctx context.Context, req *ListParkingLotsRequest) (*ParkingLotListResponse, error)
	Update(ctx context.Context, req *UpdateParkingLotRequest) (*domain.ParkingLot, error)
	Delete(ctx context.Context, tenantID, id string) error
	GetStats(ctx context.Context, tenantID string) (*ParkingLotStatsResponse, error)
}

// LaneService 出入口配置服务
type LaneService interface {
	GetLaneConfig(ctx context.Context, tenantID, parkingLotID string) (*LaneConfigResponse, error)
	UpdateLanes(ctx context.Context, tenantID string, req *UpdateLanesRequest) (*UpdateLanesResponse, error)
}

type LaneConfigResponse struct {
	Lanes            []*domain.LaneWithDevice
	AvailableDevices []*AvailableDevice
}

type AvailableDevice struct {
	ID     string
	Name   string
	Status string
}

type UpdateLanesRequest struct {
	ParkingLotID string
	Lanes        []LaneInput
}

type LaneInput struct {
	ID       *string
	Name     string
	Type     domain.LaneType
	DeviceID *string
}

type UpdateLanesResponse struct {
	Lanes []*domain.LaneWithDevice
}
