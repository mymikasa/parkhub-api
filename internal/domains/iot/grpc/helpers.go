package grpc

import (
	"context"
	"time"

	"github.com/parkhub/api/internal/domains/iot/domain"
	iotv1 "github.com/parkhub/api/internal/gen/api/proto/iot/v1"
	"github.com/parkhub/api/pkg/identityctx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func deviceTypeToProto(dt domain.DeviceType) iotv1.DeviceType {
	switch dt {
	case domain.DeviceTypeIntegrated:
		return iotv1.DeviceType_DEVICE_TYPE_INTEGRATED
	case domain.DeviceTypeCameraOnly:
		return iotv1.DeviceType_DEVICE_TYPE_CAMERA_ONLY
	case domain.DeviceTypeBarrierOnly:
		return iotv1.DeviceType_DEVICE_TYPE_BARRIER_ONLY
	default:
		return iotv1.DeviceType_DEVICE_TYPE_UNSPECIFIED
	}
}

func deviceTypeFromProto(pt iotv1.DeviceType) domain.DeviceType {
	switch pt {
	case iotv1.DeviceType_DEVICE_TYPE_INTEGRATED:
		return domain.DeviceTypeIntegrated
	case iotv1.DeviceType_DEVICE_TYPE_CAMERA_ONLY:
		return domain.DeviceTypeCameraOnly
	case iotv1.DeviceType_DEVICE_TYPE_BARRIER_ONLY:
		return domain.DeviceTypeBarrierOnly
	default:
		return ""
	}
}

func deviceStatusToProto(s domain.DeviceStatus) iotv1.DeviceStatus {
	switch s {
	case domain.DeviceStatusPending:
		return iotv1.DeviceStatus_DEVICE_STATUS_PENDING
	case domain.DeviceStatusActive:
		return iotv1.DeviceStatus_DEVICE_STATUS_ACTIVE
	case domain.DeviceStatusOffline:
		return iotv1.DeviceStatus_DEVICE_STATUS_OFFLINE
	case domain.DeviceStatusDisabled:
		return iotv1.DeviceStatus_DEVICE_STATUS_DISABLED
	default:
		return iotv1.DeviceStatus_DEVICE_STATUS_UNSPECIFIED
	}
}

func deviceStatusFromProto(ps iotv1.DeviceStatus) domain.DeviceStatus {
	switch ps {
	case iotv1.DeviceStatus_DEVICE_STATUS_PENDING:
		return domain.DeviceStatusPending
	case iotv1.DeviceStatus_DEVICE_STATUS_ACTIVE:
		return domain.DeviceStatusActive
	case iotv1.DeviceStatus_DEVICE_STATUS_OFFLINE:
		return domain.DeviceStatusOffline
	case iotv1.DeviceStatus_DEVICE_STATUS_DISABLED:
		return domain.DeviceStatusDisabled
	default:
		return ""
	}
}

func toProtoDevice(d *domain.Device) *iotv1.Device {
	if d == nil {
		return nil
	}
	pb := &iotv1.Device{
		Id:              d.ID,
		TenantId:        d.TenantID,
		Name:            d.Name,
		Type:            deviceTypeToProto(d.Type),
		Status:          deviceStatusToProto(d.Status),
		FirmwareVersion: d.FirmwareVersion,
		CreatedAt:       timeToProto(d.CreatedAt),
		UpdatedAt:       timeToProto(d.UpdatedAt),
	}
	if d.LastHeartbeat != nil {
		pb.LastHeartbeat = timeToProto(*d.LastHeartbeat)
	}
	if d.ParkingLotID != nil {
		pb.ParkingLotId = d.ParkingLotID
	}
	if d.GateID != nil {
		pb.GateId = d.GateID
	}
	return pb
}

func timeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func extractTenantID(ctx context.Context) (string, error) {
	tid := identityctx.TenantID(ctx)
	if tid == "" {
		return "", status.Error(codes.Unauthenticated, "missing x-tenant-id")
	}
	return tid, nil
}
