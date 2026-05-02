package domain

import "github.com/parkhub/api/internal/domains/parking/errs"

type LotType string

const (
	LotTypeUnderground LotType = "underground"
	LotTypeGround      LotType = "ground"
	LotTypeStereo      LotType = "stereo"
)

type ParkingLotStatus string

const (
	ParkingLotStatusActive   ParkingLotStatus = "active"
	ParkingLotStatusInactive ParkingLotStatus = "inactive"
)

type ParkingLot struct {
	ID              string
	TenantID        string
	Name            string
	Address         string
	TotalSpaces     int
	AvailableSpaces int
	LotType         LotType
	Status          ParkingLotStatus
	EntryCount      int
	ExitCount       int
	CreatedAt       int64
	UpdatedAt       int64
}

func NewParkingLot(name, address string, totalSpaces int, lotType LotType) *ParkingLot {
	return &ParkingLot{
		Name:            name,
		Address:         address,
		TotalSpaces:     totalSpaces,
		AvailableSpaces: totalSpaces,
		LotType:         lotType,
		Status:          ParkingLotStatusActive,
	}
}

func (l *ParkingLot) IsActive() bool {
	return l.Status == ParkingLotStatusActive
}

func (l *ParkingLot) Deactivate() error {
	if l.Status != ParkingLotStatusActive {
		return errs.ErrParkingLotInvalidStatus
	}
	l.Status = ParkingLotStatusInactive
	return nil
}

func (l *ParkingLot) Activate() error {
	if l.Status != ParkingLotStatusInactive {
		return errs.ErrParkingLotInvalidStatus
	}
	l.Status = ParkingLotStatusActive
	return nil
}
