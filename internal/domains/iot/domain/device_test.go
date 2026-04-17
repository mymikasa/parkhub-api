package domain

import (
	"testing"

	"github.com/parkhub/api/internal/domains/iot/errs"
	"github.com/stretchr/testify/assert"
)

func newTestDevice() *Device {
	return NewDevice("PH-DEV-001", "测试设备", DeviceTypeIntegrated, "v1.0.0")
}

func TestNewDevice(t *testing.T) {
	d := newTestDevice()
	assert.Equal(t, "PH-DEV-001", d.ID)
	assert.Equal(t, "测试设备", d.Name)
	assert.Equal(t, DeviceTypeIntegrated, d.Type)
	assert.Equal(t, DeviceStatusPending, d.Status)
	assert.Equal(t, "v1.0.0", d.FirmwareVersion)
	assert.Nil(t, d.ParkingLotID)
	assert.Nil(t, d.GateID)
}

func TestDevice_Activate(t *testing.T) {
	d := newTestDevice()
	assert.Equal(t, DeviceStatusPending, d.Status)

	err := d.Activate()
	assert.NoError(t, err)
	assert.Equal(t, DeviceStatusActive, d.Status)
}

func TestDevice_Activate_FromOffline(t *testing.T) {
	d := newTestDevice()
	d.Activate()
	d.GoOffline()

	err := d.Activate()
	assert.NoError(t, err)
	assert.Equal(t, DeviceStatusActive, d.Status)
}

func TestDevice_Activate_FromDisabled(t *testing.T) {
	d := newTestDevice()
	d.Disable()

	err := d.Activate()
	assert.Error(t, err)
}

func TestDevice_GoOffline(t *testing.T) {
	d := newTestDevice()
	d.Activate()

	err := d.GoOffline()
	assert.NoError(t, err)
	assert.Equal(t, DeviceStatusOffline, d.Status)
}

func TestDevice_GoOffline_FromPending(t *testing.T) {
	d := newTestDevice()
	err := d.GoOffline()
	assert.Error(t, err)
}

func TestDevice_Disable(t *testing.T) {
	d := newTestDevice()

	err := d.Disable()
	assert.NoError(t, err)
	assert.Equal(t, DeviceStatusDisabled, d.Status)
}

func TestDevice_Disable_AlreadyDisabled(t *testing.T) {
	d := newTestDevice()
	d.Disable()

	err := d.Disable()
	assert.Error(t, err)
}

func TestDevice_Enable(t *testing.T) {
	d := newTestDevice()
	d.Disable()

	err := d.Enable()
	assert.NoError(t, err)
	assert.Equal(t, DeviceStatusActive, d.Status)
}

func TestDevice_Enable_NotDisabled(t *testing.T) {
	d := newTestDevice()

	err := d.Enable()
	assert.Error(t, err)
}

func TestDevice_Bind(t *testing.T) {
	d := newTestDevice()

	err := d.Bind("lot-1", "gate-1")
	assert.NoError(t, err)
	assert.Equal(t, "lot-1", *d.ParkingLotID)
	assert.Equal(t, "gate-1", *d.GateID)
	assert.Equal(t, DeviceStatusActive, d.Status)
}

func TestDevice_Bind_AlreadyBound(t *testing.T) {
	d := newTestDevice()
	d.Bind("lot-1", "gate-1")

	err := d.Bind("lot-2", "gate-2")
	assert.Error(t, err)
}

func TestDevice_Unbind(t *testing.T) {
	d := newTestDevice()
	d.Bind("lot-1", "gate-1")

	err := d.Unbind()
	assert.NoError(t, err)
	assert.Nil(t, d.ParkingLotID)
	assert.Nil(t, d.GateID)
}

func TestDevice_Unbind_NotBound(t *testing.T) {
	d := newTestDevice()

	err := d.Unbind()
	assert.ErrorIs(t, err, errs.ErrDeviceNotBound)
}

func TestDevice_IsBound(t *testing.T) {
	d := newTestDevice()
	assert.False(t, d.IsBound())

	d.Bind("lot-1", "gate-1")
	assert.True(t, d.IsBound())
}

func TestDevice_UpdateName(t *testing.T) {
	d := newTestDevice()
	d.UpdateName("新名称")
	assert.Equal(t, "新名称", d.Name)
}
