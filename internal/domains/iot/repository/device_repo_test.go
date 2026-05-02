package repository

import (
	"context"
	"testing"

	"github.com/parkhub/api/internal/domains/iot/domain"
	"github.com/parkhub/api/internal/domains/iot/errs"
	"github.com/parkhub/api/internal/domains/iot/repository/dao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDeviceRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dao.Device{}))
	return db
}

func newRepoTestDevice(id string, status domain.DeviceStatus) *domain.Device {
	return &domain.Device{
		ID:              id,
		TenantID:        "tenant-1",
		Name:            "test-device-" + id,
		Type:            domain.DeviceTypeIntegrated,
		Status:          status,
		FirmwareVersion: "v1.0.0",
	}
}

func TestDeviceRepo_Transaction_RollsBackOnError(t *testing.T) {
	db := setupDeviceRepoTestDB(t)
	repo := NewDeviceRepo(dao.NewDeviceDAO(db), db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newRepoTestDevice("DEV-001", domain.DeviceStatusActive)))
	require.NoError(t, repo.Create(ctx, newRepoTestDevice("DEV-002", domain.DeviceStatusActive)))

	err := repo.Transaction(ctx, func(txRepo DeviceRepo) error {
		device, err := txRepo.GetByID(ctx, "tenant-1", "DEV-001")
		require.NoError(t, err)

		require.NoError(t, device.Disable())
		require.NoError(t, txRepo.Update(ctx, device))

		_, err = txRepo.GetByID(ctx, "tenant-1", "DEV-404")
		return err
	})
	require.ErrorIs(t, err, errs.ErrDeviceNotFound)

	device, err := repo.GetByID(ctx, "tenant-1", "DEV-001")
	require.NoError(t, err)
	assert.Equal(t, domain.DeviceStatusActive, device.Status)

	other, err := repo.GetByID(ctx, "tenant-1", "DEV-002")
	require.NoError(t, err)
	assert.Equal(t, domain.DeviceStatusActive, other.Status)
}
