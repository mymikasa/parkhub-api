package grpc

import (
	"context"

	"github.com/parkhub/api/services/parking/internal/domain"
	parkingv1 "github.com/parkhub/api/services/parking/internal/gen/api/proto/parking/v1"
	"github.com/parkhub/api/services/parking/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type LaneGRPCServer struct {
	parkingv1.UnimplementedLaneServiceServer
	laneSvc service.LaneService
}

func NewLaneGRPCServer(laneSvc service.LaneService) *LaneGRPCServer {
	return &LaneGRPCServer{laneSvc: laneSvc}
}

func (s *LaneGRPCServer) GetLaneConfig(ctx context.Context, req *parkingv1.GetLaneConfigRequest) (*parkingv1.GetLaneConfigResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetParkingLotId() == "" {
		return nil, status.Error(codes.InvalidArgument, "parking_lot_id is required")
	}

	resp, err := s.laneSvc.GetLaneConfig(ctx, tenantID, req.GetParkingLotId())
	if err != nil {
		return nil, toGRPCError(err)
	}

	pbLanes := make([]*parkingv1.Lane, len(resp.Lanes))
	for i, l := range resp.Lanes {
		pbLanes[i] = toProtoLane(&l.Lane, l.Device)
	}

	pbDevices := make([]*parkingv1.DeviceOption, len(resp.AvailableDevices))
	for i, d := range resp.AvailableDevices {
		pbDevices[i] = &parkingv1.DeviceOption{
			DeviceId: d.ID,
			Name:     d.Name,
			Status:   d.Status,
		}
	}

	return &parkingv1.GetLaneConfigResponse{
		Lanes:            pbLanes,
		AvailableDevices: pbDevices,
	}, nil
}

func (s *LaneGRPCServer) UpdateLanes(ctx context.Context, req *parkingv1.UpdateLanesRequest) (*parkingv1.UpdateLanesResponse, error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetParkingLotId() == "" {
		return nil, status.Error(codes.InvalidArgument, "parking_lot_id is required")
	}

	lanes := make([]service.LaneInput, len(req.Lanes))
	for i, input := range req.Lanes {
		lanes[i] = service.LaneInput{
			Name: input.GetName(),
			Type: laneTypeFromProto(input.GetLaneType()),
		}
		if input.GetLaneId() != "" {
			id := input.GetLaneId()
			lanes[i].ID = &id
		}
		if input.GetDeviceId() != "" {
			did := input.GetDeviceId()
			lanes[i].DeviceID = &did
		}
	}

	resp, err := s.laneSvc.UpdateLanes(ctx, tenantID, &service.UpdateLanesRequest{
		ParkingLotID: req.GetParkingLotId(),
		Lanes:        lanes,
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	pbLanes := make([]*parkingv1.Lane, len(resp.Lanes))
	for i, l := range resp.Lanes {
		pbLanes[i] = toProtoLane(&l.Lane, l.Device)
	}

	return &parkingv1.UpdateLanesResponse{Lanes: pbLanes}, nil
}

func toProtoLane(l *domain.Lane, dev *domain.LaneDeviceInfo) *parkingv1.Lane {
	if l == nil {
		return nil
	}
	pb := &parkingv1.Lane{
		LaneId:       l.ID,
		ParkingLotId: l.ParkingLotID,
		Name:         l.Name,
		LaneType:     laneTypeToProto(l.Type),
		CreatedAt:    toTimestamp(l.CreatedAt),
		UpdatedAt:    toTimestamp(l.UpdatedAt),
	}
	if l.DeviceID != nil {
		pb.DeviceId = *l.DeviceID
	}
	if dev != nil {
		pb.Device = &parkingv1.LaneDevice{
			DeviceId: dev.ID,
			Name:     dev.Name,
			Status:   dev.Status,
		}
		if dev.LastHeartbeat != nil {
			pb.Device.LastHeartbeat = timestamppb.New(*dev.LastHeartbeat)
		}
	}
	return pb
}

func laneTypeToProto(t domain.LaneType) parkingv1.LaneType {
	switch t {
	case domain.LaneTypeEntry:
		return parkingv1.LaneType_LANE_TYPE_ENTRY
	case domain.LaneTypeExit:
		return parkingv1.LaneType_LANE_TYPE_EXIT
	default:
		return parkingv1.LaneType_LANE_TYPE_UNSPECIFIED
	}
}

func laneTypeFromProto(t parkingv1.LaneType) domain.LaneType {
	switch t {
	case parkingv1.LaneType_LANE_TYPE_ENTRY:
		return domain.LaneTypeEntry
	case parkingv1.LaneType_LANE_TYPE_EXIT:
		return domain.LaneTypeExit
	default:
		return ""
	}
}
