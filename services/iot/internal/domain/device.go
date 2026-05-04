package domain

import (
	"errors"
	"time"

	"github.com/parkhub/api/services/iot/internal/errs"
)

type DeviceType string

const (
	DeviceTypeIntegrated  DeviceType = "integrated"
	DeviceTypeCameraOnly  DeviceType = "camera_only"
	DeviceTypeBarrierOnly DeviceType = "barrier_only"
)

type DeviceStatus string

const (
	DeviceStatusPending  DeviceStatus = "pending"
	DeviceStatusActive   DeviceStatus = "active"
	DeviceStatusOffline  DeviceStatus = "offline"
	DeviceStatusDisabled DeviceStatus = "disabled"
)

type Device struct {
	ID              string
	TenantID        string
	Name            string
	Type            DeviceType
	Status          DeviceStatus
	FirmwareVersion string
	LastHeartbeat   *time.Time
	ParkingLotID    *string
	GateID          *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewDevice(id, name string, deviceType DeviceType, firmwareVersion string) *Device {
	now := time.Now()
	return &Device{
		ID:              id,
		Name:            name,
		Type:            deviceType,
		Status:          DeviceStatusPending,
		FirmwareVersion: firmwareVersion,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (d *Device) Activate() error {
	if d.Status != DeviceStatusPending && d.Status != DeviceStatusOffline {
		return errors.New("设备状态不允许激活")
	}
	d.Status = DeviceStatusActive
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Device) GoOffline() error {
	if d.Status != DeviceStatusActive {
		return errors.New("只有已激活的设备可以离线")
	}
	d.Status = DeviceStatusOffline
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Device) Disable() error {
	if d.Status == DeviceStatusDisabled {
		return errors.New("设备已是禁用状态")
	}
	d.Status = DeviceStatusDisabled
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Device) Enable() error {
	if d.Status != DeviceStatusDisabled {
		return errors.New("只有已禁用的设备可以启用")
	}
	d.Status = DeviceStatusActive
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Device) Bind(parkingLotID, gateID string) error {
	if d.ParkingLotID != nil {
		return errors.New("设备已绑定，请先解绑")
	}
	d.ParkingLotID = &parkingLotID
	d.GateID = &gateID
	if d.Status == DeviceStatusPending {
		d.Status = DeviceStatusActive
	}
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Device) Unbind() error {
	if d.ParkingLotID == nil {
		return errs.ErrDeviceNotBound
	}
	d.ParkingLotID = nil
	d.GateID = nil
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Device) UpdateName(name string) {
	d.Name = name
	d.UpdatedAt = time.Now()
}

func (d *Device) IsBound() bool {
	return d.ParkingLotID != nil
}
