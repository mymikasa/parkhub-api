package repository

import (
	"context"
	"testing"

	"github.com/parkhub/api/internal/domains/parking/domain"
	"github.com/parkhub/api/internal/domains/parking/repository/dao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRepoTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&dao.ParkingLot{}))
	return db
}

func TestParkingLotRepo_Create(t *testing.T) {
	db := setupRepoTestDB(t)
	lotDAO := dao.NewParkingLotDAO(db)
	repo := NewParkingLotRepo(lotDAO)
	ctx := context.Background()

	lot := &domain.ParkingLot{
		ID:              "lot-1",
		TenantID:        "tenant-1",
		Name:            "Test Lot",
		Address:         "123 Test St",
		TotalSpaces:     200,
		AvailableSpaces: 200,
		LotType:         domain.LotTypeGround,
		Status:          domain.ParkingLotStatusActive,
	}
	err := repo.Create(ctx, lot)
	assert.NoError(t, err)
}

func TestParkingLotRepo_GetByID(t *testing.T) {
	db := setupRepoTestDB(t)
	lotDAO := dao.NewParkingLotDAO(db)
	repo := NewParkingLotRepo(lotDAO)
	ctx := context.Background()

	lot := &domain.ParkingLot{
		ID:              "lot-1",
		TenantID:        "tenant-1",
		Name:            "Test Lot",
		Address:         "123 Test St",
		TotalSpaces:     200,
		AvailableSpaces: 180,
		LotType:         domain.LotTypeUnderground,
		Status:          domain.ParkingLotStatusActive,
	}
	require.NoError(t, repo.Create(ctx, lot))

	found, err := repo.GetByID(ctx, "tenant-1", "lot-1")
	assert.NoError(t, err)
	assert.Equal(t, "lot-1", found.ID)
	assert.Equal(t, "tenant-1", found.TenantID)
	assert.Equal(t, domain.LotTypeUnderground, found.LotType)
	assert.Equal(t, domain.ParkingLotStatusActive, found.Status)
	assert.Equal(t, 200, found.TotalSpaces)
	assert.Equal(t, 180, found.AvailableSpaces)
}

func TestParkingLotRepo_GetByID_Conversion(t *testing.T) {
	db := setupRepoTestDB(t)
	lotDAO := dao.NewParkingLotDAO(db)
	repo := NewParkingLotRepo(lotDAO)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &domain.ParkingLot{
		ID:       "lot-1",
		TenantID: "tenant-1",
		Name:     "Test",
		LotType:  domain.LotTypeStereo,
		Status:   domain.ParkingLotStatusInactive,
	}))

	found, err := repo.GetByID(ctx, "tenant-1", "lot-1")
	assert.NoError(t, err)
	assert.Equal(t, domain.LotTypeStereo, found.LotType)
	assert.Equal(t, domain.ParkingLotStatusInactive, found.Status)
}

func TestParkingLotRepo_List(t *testing.T) {
	db := setupRepoTestDB(t)
	lotDAO := dao.NewParkingLotDAO(db)
	repo := NewParkingLotRepo(lotDAO)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &domain.ParkingLot{
		ID: "lot-1", TenantID: "tenant-1", Name: "A", Status: domain.ParkingLotStatusActive,
	}))
	require.NoError(t, repo.Create(ctx, &domain.ParkingLot{
		ID: "lot-2", TenantID: "tenant-1", Name: "B", Status: domain.ParkingLotStatusInactive,
	}))

	filter := ParkingLotFilter{TenantID: "tenant-1", Status: domain.ParkingLotStatusActive}
	lots, total, err := repo.List(ctx, filter, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, lots, 1)
	assert.Equal(t, domain.ParkingLotStatusActive, lots[0].Status)
}

func TestParkingLotRepo_Update(t *testing.T) {
	db := setupRepoTestDB(t)
	lotDAO := dao.NewParkingLotDAO(db)
	repo := NewParkingLotRepo(lotDAO)
	ctx := context.Background()

	lot := &domain.ParkingLot{
		ID:              "lot-1",
		TenantID:        "tenant-1",
		Name:            "Original",
		Address:         "Old Address",
		TotalSpaces:     100,
		AvailableSpaces: 100,
		LotType:         domain.LotTypeGround,
		Status:          domain.ParkingLotStatusActive,
	}
	require.NoError(t, repo.Create(ctx, lot))

	lot.Address = "New Address"
	err := repo.Update(ctx, lot)
	assert.NoError(t, err)

	found, _ := repo.GetByID(ctx, "tenant-1", "lot-1")
	assert.Equal(t, "New Address", found.Address)
}

func TestParkingLotRepo_Delete(t *testing.T) {
	db := setupRepoTestDB(t)
	lotDAO := dao.NewParkingLotDAO(db)
	repo := NewParkingLotRepo(lotDAO)
	ctx := context.Background()

	lot := &domain.ParkingLot{
		ID: "lot-1", TenantID: "tenant-1", Name: "Test", LotType: domain.LotTypeGround, Status: domain.ParkingLotStatusActive,
	}
	require.NoError(t, repo.Create(ctx, lot))

	err := repo.Delete(ctx, "tenant-1", "lot-1")
	assert.NoError(t, err)

	_, err = repo.GetByID(ctx, "tenant-1", "lot-1")
	assert.Error(t, err)
}

func TestParkingLotRepo_SumStats(t *testing.T) {
	db := setupRepoTestDB(t)
	lotDAO := dao.NewParkingLotDAO(db)
	repo := NewParkingLotRepo(lotDAO)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &domain.ParkingLot{
		ID: "lot-1", TenantID: "tenant-1", Name: "A", TotalSpaces: 200, AvailableSpaces: 150, LotType: domain.LotTypeGround, Status: domain.ParkingLotStatusActive,
	}))
	require.NoError(t, repo.Create(ctx, &domain.ParkingLot{
		ID: "lot-2", TenantID: "tenant-1", Name: "B", TotalSpaces: 100, AvailableSpaces: 80, LotType: domain.LotTypeGround, Status: domain.ParkingLotStatusActive,
	}))

	total, available, err := repo.SumStats(ctx, "tenant-1")
	assert.NoError(t, err)
	assert.Equal(t, int64(300), total)
	assert.Equal(t, int64(230), available)
}
