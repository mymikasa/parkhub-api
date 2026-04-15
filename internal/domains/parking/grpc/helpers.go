package grpc

import (
	"github.com/parkhub/api/internal/domains/parking/domain"
	parkingv1 "github.com/parkhub/api/internal/gen/api/proto/parking/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoParkingLot(l *domain.ParkingLot) *parkingv1.ParkingLot {
	if l == nil {
		return nil
	}
	return &parkingv1.ParkingLot{
		Id:              l.ID,
		TenantId:        l.TenantID,
		Name:            l.Name,
		Address:         l.Address,
		TotalSpaces:     int32(l.TotalSpaces),
		AvailableSpaces: int32(l.AvailableSpaces),
		LotType:         domainLotTypeToProto(l.LotType),
		Status:          domainStatusToProto(l.Status),
		CreatedAt:       toTimestamp(l.CreatedAt),
		UpdatedAt:       toTimestamp(l.UpdatedAt),
	}
}

func domainLotTypeToProto(lt domain.LotType) parkingv1.LotType {
	switch lt {
	case domain.LotTypeUnderground:
		return parkingv1.LotType_LOT_TYPE_UNDERGROUND
	case domain.LotTypeGround:
		return parkingv1.LotType_LOT_TYPE_GROUND
	case domain.LotTypeStereo:
		return parkingv1.LotType_LOT_TYPE_STEREO
	default:
		return parkingv1.LotType_LOT_TYPE_UNSPECIFIED
	}
}

func domainLotTypeFromProto(lt parkingv1.LotType) domain.LotType {
	switch lt {
	case parkingv1.LotType_LOT_TYPE_UNDERGROUND:
		return domain.LotTypeUnderground
	case parkingv1.LotType_LOT_TYPE_GROUND:
		return domain.LotTypeGround
	case parkingv1.LotType_LOT_TYPE_STEREO:
		return domain.LotTypeStereo
	default:
		return domain.LotTypeGround
	}
}

func domainStatusToProto(s domain.ParkingLotStatus) parkingv1.ParkingLotStatus {
	switch s {
	case domain.ParkingLotStatusActive:
		return parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_ACTIVE
	case domain.ParkingLotStatusInactive:
		return parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_INACTIVE
	default:
		return parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_UNSPECIFIED
	}
}

func domainStatusFromProto(s parkingv1.ParkingLotStatus) domain.ParkingLotStatus {
	switch s {
	case parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_ACTIVE:
		return domain.ParkingLotStatusActive
	case parkingv1.ParkingLotStatus_PARKING_LOT_STATUS_INACTIVE:
		return domain.ParkingLotStatusInactive
	default:
		return domain.ParkingLotStatusActive
	}
}

func toTimestamp(millis int64) *timestamppb.Timestamp {
	if millis == 0 {
		return nil
	}
	seconds := millis / 1000
	nanos := int32((millis % 1000) * 1e6)
	return &timestamppb.Timestamp{Seconds: seconds, Nanos: nanos}
}
