package grpc

import (
	"context"

	"github.com/parkhub/api/internal/domains/parking/errs"
	"github.com/parkhub/api/internal/domains/parking/service"
	commonv1 "github.com/parkhub/api/internal/gen/api/proto/common/v1"
	parkingv1 "github.com/parkhub/api/internal/gen/api/proto/parking/v1"
	"github.com/parkhub/api/pkg/grpcutil"
	"github.com/parkhub/api/pkg/identityctx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ParkingLotGRPCServer struct {
	parkingv1.UnimplementedParkingLotServiceServer
	lotSvc service.ParkingLotService
}

func NewParkingLotGRPCServer(svc service.ParkingLotService) *ParkingLotGRPCServer {
	return &ParkingLotGRPCServer{lotSvc: svc}
}

var errorMappings = []grpcutil.ErrorMapping{
	{Target: errs.ErrParkingLotNotFound, Code: codes.NotFound},
	{Target: errs.ErrParkingLotNameDuplicate, Code: codes.AlreadyExists},
	{Target: errs.ErrParkingLotInvalidStatus, Code: codes.InvalidArgument},
	{Target: errs.ErrParkingLotInvalidCapacity, Code: codes.InvalidArgument},
	{Target: errs.ErrLaneNotFound, Code: codes.NotFound},
	{Target: errs.ErrLaneNameDuplicate, Code: codes.AlreadyExists},
}

func toGRPCError(err error) error {
	return grpcutil.ToGRPCError(err, errorMappings)
}

func extractTenantID(ctx context.Context) (string, error) {
	tid := identityctx.TenantID(ctx)
	if tid == "" {
		return "", status.Error(codes.Unauthenticated, "missing x-tenant-id")
	}
	return tid, nil
}

func (s *ParkingLotGRPCServer) CreateParkingLot(ctx context.Context, req *parkingv1.CreateParkingLotRequest) (*parkingv1.CreateParkingLotResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.GetAddress() == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}
	if req.GetTotalSpaces() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "total_spaces must be positive")
	}

	lot, err := s.lotSvc.Create(ctx, &service.CreateParkingLotRequest{
		TenantID:    tenantID,
		Name:        req.GetName(),
		Address:     req.GetAddress(),
		TotalSpaces: int(req.GetTotalSpaces()),
		LotType:     domainLotTypeFromProto(req.GetLotType()),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &parkingv1.CreateParkingLotResponse{ParkingLot: toProtoParkingLot(lot)}, nil
}

func (s *ParkingLotGRPCServer) GetParkingLot(ctx context.Context, req *parkingv1.GetParkingLotRequest) (*parkingv1.GetParkingLotResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	lot, err := s.lotSvc.GetByID(ctx, tenantID, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &parkingv1.GetParkingLotResponse{ParkingLot: toProtoParkingLot(lot)}, nil
}

func (s *ParkingLotGRPCServer) ListParkingLots(ctx context.Context, req *parkingv1.ListParkingLotsRequest) (*parkingv1.ListParkingLotsResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	page := int32(1)
	pageSize := int32(20)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PageSize > 0 {
			pageSize = req.Pagination.PageSize
		}
	}

	resp, err := s.lotSvc.List(ctx, &service.ListParkingLotsRequest{
		TenantID: tenantID,
		Status:   domainStatusFromProto(req.GetStatus()),
		LotType:  domainLotTypeFromProto(req.GetLotType()),
		Keyword:  req.GetKeyword(),
		Page:     int(page),
		PageSize: int(pageSize),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	lots := make([]*parkingv1.ParkingLot, 0, len(resp.ParkingLots))
	for _, l := range resp.ParkingLots {
		lots = append(lots, toProtoParkingLot(l))
	}

	return &parkingv1.ListParkingLotsResponse{
		ParkingLots: lots,
		Pagination: &commonv1.PaginationResponse{
			Page:       int32(resp.Page),
			PageSize:   int32(resp.PageSize),
			Total:      resp.Total,
			TotalPages: int32(resp.TotalPages),
		},
	}, nil
}

func (s *ParkingLotGRPCServer) UpdateParkingLot(ctx context.Context, req *parkingv1.UpdateParkingLotRequest) (*parkingv1.UpdateParkingLotResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	svcReq := &service.UpdateParkingLotRequest{
		ID:       req.GetId(),
		TenantID: tenantID,
	}
	if req.Name != nil {
		svcReq.Name = req.Name
	}
	if req.Address != nil {
		svcReq.Address = req.Address
	}
	if req.TotalSpaces != nil {
		ts := int(*req.TotalSpaces)
		svcReq.TotalSpaces = &ts
	}
	if req.LotType != nil {
		lt := domainLotTypeFromProto(*req.LotType)
		svcReq.LotType = &lt
	}
	if req.Status != nil {
		st := domainStatusFromProto(*req.Status)
		svcReq.Status = &st
	}

	lot, err := s.lotSvc.Update(ctx, svcReq)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &parkingv1.UpdateParkingLotResponse{ParkingLot: toProtoParkingLot(lot)}, nil
}

func (s *ParkingLotGRPCServer) DeleteParkingLot(ctx context.Context, req *parkingv1.DeleteParkingLotRequest) (*parkingv1.DeleteParkingLotResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	if err := s.lotSvc.Delete(ctx, tenantID, req.GetId()); err != nil {
		return nil, toGRPCError(err)
	}
	return &parkingv1.DeleteParkingLotResponse{}, nil
}

func (s *ParkingLotGRPCServer) GetParkingLotStats(ctx context.Context, req *parkingv1.GetParkingLotStatsRequest) (*parkingv1.GetParkingLotStatsResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	stats, err := s.lotSvc.GetStats(ctx, tenantID)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &parkingv1.GetParkingLotStatsResponse{
		TotalSpaces:      stats.TotalSpaces,
		AvailableSpaces:  stats.AvailableSpaces,
		OccupiedVehicles: stats.OccupiedVehicles,
		TotalGates:       stats.TotalGates,
	}, nil
}
