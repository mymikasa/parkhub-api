package dao

import (
	"context"
	"testing"

	"github.com/parkhub/api/internal/domains/iot/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIOTTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Device{}))
	return db
}

func newTestDevice(id, name string) *Device {
	return &Device{
		ID:              id,
		TenantID:        "tenant-1",
		Name:            name,
		Type:            "integrated",
		Status:          "pending",
		FirmwareVersion: "v1.0.0",
	}
}

func TestDeviceDAO_Insert(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	err := dao.Insert(ctx, newTestDevice("DEV-001", "测试设备"))
	assert.NoError(t, err)
}

func TestDeviceDAO_Insert_DuplicateID(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "设备1")))
	err := dao.Insert(ctx, newTestDevice("DEV-001", "设备2"))
	assert.ErrorIs(t, err, errs.ErrDeviceIDDuplicate)
}

func TestDeviceDAO_FindByID(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "测试设备")))

	device, err := dao.FindByID(ctx, "tenant-1", "DEV-001")
	assert.NoError(t, err)
	assert.Equal(t, "测试设备", device.Name)
}

func TestDeviceDAO_FindByID_NotFound(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	_, err := dao.FindByID(ctx, "tenant-1", "BAD-ID")
	assert.ErrorIs(t, err, errs.ErrDeviceNotFound)
}

func TestDeviceDAO_FindByID_WrongTenant(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "测试设备")))

	_, err := dao.FindByID(ctx, "wrong-tenant", "DEV-001")
	assert.ErrorIs(t, err, errs.ErrDeviceNotFound)
}

func TestDeviceDAO_FindAll(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "设备1")))
	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-002", "设备2")))

	devices, total, err := dao.FindAll(ctx, DeviceFilter{TenantID: "tenant-1"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, devices, 2)
}

func TestDeviceDAO_FindAll_FilterByStatus(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	d1 := newTestDevice("DEV-001", "设备1")
	d1.Status = "active"
	require.NoError(t, dao.Insert(ctx, d1))

	d2 := newTestDevice("DEV-002", "设备2")
	d2.Status = "offline"
	require.NoError(t, dao.Insert(ctx, d2))

	devices, total, err := dao.FindAll(ctx, DeviceFilter{TenantID: "tenant-1", Status: "active"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, devices, 1)
	assert.Equal(t, "active", devices[0].Status)
}

func TestDeviceDAO_FindAll_KeywordSearch(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "入口摄像头")))
	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-002", "出口道闸")))

	devices, total, err := dao.FindAll(ctx, DeviceFilter{TenantID: "tenant-1", Keyword: "DEV-001"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, devices, 1)
	assert.Equal(t, "DEV-001", devices[0].ID)
}

func TestDeviceDAO_Update(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "旧名称")))

	device, err := dao.FindByID(ctx, "tenant-1", "DEV-001")
	require.NoError(t, err)
	device.Name = "新名称"
	device.Status = "active"

	err = dao.Update(ctx, device)
	assert.NoError(t, err)

	updated, err := dao.FindByID(ctx, "tenant-1", "DEV-001")
	assert.NoError(t, err)
	assert.Equal(t, "新名称", updated.Name)
	assert.Equal(t, "active", updated.Status)
}

func TestDeviceDAO_Update_NotFound(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	err := dao.Update(ctx, &Device{ID: "BAD-ID", TenantID: "tenant-1", Name: "test"})
	assert.ErrorIs(t, err, errs.ErrDeviceNotFound)
}

func TestDeviceDAO_Delete(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "测试")))

	err := dao.Delete(ctx, "tenant-1", "DEV-001")
	assert.NoError(t, err)

	_, err = dao.FindByID(ctx, "tenant-1", "DEV-001")
	assert.ErrorIs(t, err, errs.ErrDeviceNotFound)
}

func TestDeviceDAO_Delete_NotFound(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	err := dao.Delete(ctx, "tenant-1", "BAD-ID")
	assert.ErrorIs(t, err, errs.ErrDeviceNotFound)
}

func TestDeviceDAO_DeleteBatch(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "设备1")))
	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-002", "设备2")))
	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-003", "设备3")))

	affected, err := dao.DeleteBatch(ctx, "tenant-1", []string{"DEV-001", "DEV-003"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), affected)

	_, total, _ := dao.FindAll(ctx, DeviceFilter{TenantID: "tenant-1"}, 1, 10)
	assert.Equal(t, int64(1), total)
}

func TestDeviceDAO_CountByStatus(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	d1 := newTestDevice("DEV-001", "设备1")
	d1.Status = "active"
	require.NoError(t, dao.Insert(ctx, d1))

	d2 := newTestDevice("DEV-002", "设备2")
	d2.Status = "active"
	require.NoError(t, dao.Insert(ctx, d2))

	d3 := newTestDevice("DEV-003", "设备3")
	d3.Status = "offline"
	require.NoError(t, dao.Insert(ctx, d3))

	d4 := newTestDevice("DEV-004", "设备4")
	d4.Status = "pending"
	require.NoError(t, dao.Insert(ctx, d4))

	pending, active, offline, disabled, err := dao.CountByStatus(ctx, "tenant-1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pending)
	assert.Equal(t, int64(2), active)
	assert.Equal(t, int64(1), offline)
	assert.Equal(t, int64(0), disabled)
}

func TestDeviceDAO_UpdateStatus(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "设备1")))

	err := dao.UpdateStatus(ctx, "tenant-1", "DEV-001", "active")
	assert.NoError(t, err)

	device, _ := dao.FindByID(ctx, "tenant-1", "DEV-001")
	assert.Equal(t, "active", device.Status)
}

func TestDeviceDAO_UpdateStatusBatch(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-001", "设备1")))
	require.NoError(t, dao.Insert(ctx, newTestDevice("DEV-002", "设备2")))

	affected, err := dao.UpdateStatusBatch(ctx, "tenant-1", []string{"DEV-001", "DEV-002"}, "disabled")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), affected)
}

func TestDeviceDAO_UnbindByDeviceIDs(t *testing.T) {
	db := setupIOTTestDB(t)
	dao := NewDeviceDAO(db)
	ctx := context.Background()

	d := newTestDevice("DEV-001", "设备1")
	lotID := "lot-1"
	gateID := "gate-1"
	d.ParkingLotID = &lotID
	d.GateID = &gateID
	require.NoError(t, dao.Insert(ctx, d))

	err := dao.UnbindByDeviceIDs(ctx, "tenant-1", []string{"DEV-001"})
	assert.NoError(t, err)

	device, _ := dao.FindByID(ctx, "tenant-1", "DEV-001")
	assert.Nil(t, device.ParkingLotID)
	assert.Nil(t, device.GateID)
}
