package domain

import (
	"testing"

	"github.com/parkhub/api/internal/domains/parking/errs"
	"github.com/stretchr/testify/assert"
)

func TestNewParkingLot(t *testing.T) {
	lot := NewParkingLot("朝阳停车场", "朝阳区xxx", 200, LotTypeUnderground)

	assert.Equal(t, "朝阳停车场", lot.Name)
	assert.Equal(t, "朝阳区xxx", lot.Address)
	assert.Equal(t, 200, lot.TotalSpaces)
	assert.Equal(t, 200, lot.AvailableSpaces)
	assert.Equal(t, LotTypeUnderground, lot.LotType)
	assert.Equal(t, ParkingLotStatusActive, lot.Status)
	assert.Empty(t, lot.ID)
	assert.Empty(t, lot.TenantID)
}

func TestNewParkingLot_AvailableSpacesEqualsTotalSpaces(t *testing.T) {
	lot := NewParkingLot("test", "addr", 50, LotTypeGround)
	assert.Equal(t, lot.TotalSpaces, lot.AvailableSpaces)
}

func TestIsActive(t *testing.T) {
	lot := NewParkingLot("test", "addr", 100, LotTypeGround)
	assert.True(t, lot.IsActive())

	lot.Status = ParkingLotStatusInactive
	assert.False(t, lot.IsActive())
}

func TestDeactivate(t *testing.T) {
	lot := NewParkingLot("test", "addr", 100, LotTypeGround)
	err := lot.Deactivate()
	assert.NoError(t, err)
	assert.Equal(t, ParkingLotStatusInactive, lot.Status)
}

func TestDeactivate_AlreadyInactive(t *testing.T) {
	lot := NewParkingLot("test", "addr", 100, LotTypeGround)
	_ = lot.Deactivate()
	err := lot.Deactivate()
	assert.ErrorIs(t, err, errs.ErrParkingLotInvalidStatus)
}

func TestActivate(t *testing.T) {
	lot := NewParkingLot("test", "addr", 100, LotTypeGround)
	_ = lot.Deactivate()
	err := lot.Activate()
	assert.NoError(t, err)
	assert.Equal(t, ParkingLotStatusActive, lot.Status)
}

func TestActivate_AlreadyActive(t *testing.T) {
	lot := NewParkingLot("test", "addr", 100, LotTypeGround)
	err := lot.Activate()
	assert.ErrorIs(t, err, errs.ErrParkingLotInvalidStatus)
}
