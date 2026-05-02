package domain

import "time"

type LaneType string

const (
	LaneTypeEntry LaneType = "entry"
	LaneTypeExit  LaneType = "exit"
)

type Lane struct {
	ID           string
	TenantID     string
	ParkingLotID string
	Name         string
	Type         LaneType
	DeviceID     *string
	CreatedAt    int64
	UpdatedAt    int64
}

func NewLane(id, tenantID, parkingLotID, name string, laneType LaneType) *Lane {
	return &Lane{
		ID:           id,
		TenantID:     tenantID,
		ParkingLotID: parkingLotID,
		Name:         name,
		Type:         laneType,
		CreatedAt:    time.Now().UnixMilli(),
		UpdatedAt:    time.Now().UnixMilli(),
	}
}

func (l *Lane) SetDevice(deviceID string) {
	l.DeviceID = &deviceID
	l.UpdatedAt = time.Now().UnixMilli()
}

func (l *Lane) ClearDevice() {
	l.DeviceID = nil
	l.UpdatedAt = time.Now().UnixMilli()
}

type LaneWithDevice struct {
	Lane
	Device *LaneDeviceInfo
}

type LaneDeviceInfo struct {
	ID            string
	Name          string
	Status        string
	LastHeartbeat *time.Time
}
