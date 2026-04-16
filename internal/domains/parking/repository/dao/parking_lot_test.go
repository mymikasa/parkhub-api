package dao

import (
	"context"
	"fmt"
	"testing"

	"github.com/parkhub/api/internal/domains/parking/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupParkingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&ParkingLot{}))
	return db
}

var testLotSeq int

func newTestLot(tenantID, name string) *ParkingLot {
	testLotSeq++
	return &ParkingLot{
		ID:              fmt.Sprintf("%s-lot-%d", tenantID, testLotSeq),
		TenantID:        tenantID,
		Name:            name,
		Address:         name + " address",
		TotalSpaces:     100,
		AvailableSpaces: 100,
		LotType:         "ground",
		Status:          "active",
	}
}

func TestParkingLotDAO_Insert(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot := newTestLot("tenant-1", "朝阳停车场")
	err := dao.Insert(ctx, lot)
	assert.NoError(t, err)
}

func TestParkingLotDAO_Insert_DuplicateName(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot := newTestLot("tenant-1", "朝阳停车场")
	require.NoError(t, dao.Insert(ctx, lot))

	err := dao.Insert(ctx, &ParkingLot{
		ID:       "dup-id",
		TenantID: "tenant-1",
		Name:     "朝阳停车场",
		Address:  "other address",
		LotType:  "ground",
		Status:   "active",
	})
	assert.ErrorIs(t, err, errs.ErrParkingLotNameDuplicate)
}

func TestParkingLotDAO_Insert_DifferentTenantSameName(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot1 := newTestLot("tenant-1", "朝阳停车场")
	require.NoError(t, dao.Insert(ctx, lot1))

	lot2 := newTestLot("tenant-2", "朝阳停车场")
	err := dao.Insert(ctx, lot2)
	assert.NoError(t, err)
}

func TestParkingLotDAO_FindByID(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot := newTestLot("tenant-1", "朝阳停车场")
	require.NoError(t, dao.Insert(ctx, lot))

	found, err := dao.FindByID(ctx, "tenant-1", lot.ID)
	assert.NoError(t, err)
	assert.Equal(t, lot.Name, found.Name)
	assert.Equal(t, lot.TenantID, found.TenantID)
}

func TestParkingLotDAO_FindByID_NotFound(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	_, err := dao.FindByID(ctx, "tenant-1", "nonexistent")
	assert.ErrorIs(t, err, errs.ErrParkingLotNotFound)
}

func TestParkingLotDAO_FindByID_WrongTenant(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot := newTestLot("tenant-1", "朝阳停车场")
	require.NoError(t, dao.Insert(ctx, lot))

	_, err := dao.FindByID(ctx, "tenant-2", lot.ID)
	assert.ErrorIs(t, err, errs.ErrParkingLotNotFound)
}

func TestParkingLotDAO_FindAll(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		require.NoError(t, dao.Insert(ctx, newTestLot("tenant-1", name)))
	}

	lots, total, err := dao.FindAll(ctx, ParkingLotFilter{TenantID: "tenant-1"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, lots, 3)
}

func TestParkingLotDAO_FindAll_FilterByStatus(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot1 := newTestLot("tenant-1", "Active")
	lot1.Status = "active"
	require.NoError(t, dao.Insert(ctx, lot1))

	lot2 := newTestLot("tenant-1", "Inactive")
	lot2.Status = "inactive"
	require.NoError(t, dao.Insert(ctx, lot2))

	lots, total, err := dao.FindAll(ctx, ParkingLotFilter{TenantID: "tenant-1", Status: "active"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, lots, 1)
	assert.Equal(t, "active", lots[0].Status)
}

func TestParkingLotDAO_FindAll_FilterByLotType(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot1 := newTestLot("tenant-1", "GroundLot")
	lot1.LotType = "ground"
	require.NoError(t, dao.Insert(ctx, lot1))

	lot2 := newTestLot("tenant-1", "UndergroundLot")
	lot2.LotType = "underground"
	require.NoError(t, dao.Insert(ctx, lot2))

	lots, total, err := dao.FindAll(ctx, ParkingLotFilter{TenantID: "tenant-1", LotType: "underground"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, lots, 1)
	assert.Equal(t, "underground", lots[0].LotType)
}

func TestParkingLotDAO_FindAll_KeywordSearch(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestLot("tenant-1", "朝阳停车场")))
	require.NoError(t, dao.Insert(ctx, newTestLot("tenant-1", "海淀停车场")))

	lots, total, err := dao.FindAll(ctx, ParkingLotFilter{TenantID: "tenant-1", Keyword: "朝阳"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, lots, 1)
	assert.Contains(t, lots[0].Name, "朝阳")
}

func TestParkingLotDAO_FindAll_KeywordSearchByAddress(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot := newTestLot("tenant-1", "TestLot")
	lot.Address = "朝阳区建国路88号"
	require.NoError(t, dao.Insert(ctx, lot))

	lots, total, err := dao.FindAll(ctx, ParkingLotFilter{TenantID: "tenant-1", Keyword: "建国路"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, lots, 1)
}

func TestParkingLotDAO_FindAll_Pagination(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, dao.Insert(ctx, newTestLot("tenant-1", string(rune('A'+i)))))
	}

	lots, total, err := dao.FindAll(ctx, ParkingLotFilter{TenantID: "tenant-1"}, 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, lots, 2)
}

func TestParkingLotDAO_FindAll_TenantIsolation(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestLot("tenant-1", "Lot1")))
	require.NoError(t, dao.Insert(ctx, newTestLot("tenant-2", "Lot2")))

	lots, total, err := dao.FindAll(ctx, ParkingLotFilter{TenantID: "tenant-1"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, lots, 1)
	assert.Equal(t, "tenant-1", lots[0].TenantID)
}

func TestParkingLotDAO_Update(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot := newTestLot("tenant-1", "朝阳停车场")
	require.NoError(t, dao.Insert(ctx, lot))

	lot.Address = "更新后地址"
	err := dao.Update(ctx, lot)
	assert.NoError(t, err)

	found, _ := dao.FindByID(ctx, "tenant-1", lot.ID)
	assert.Equal(t, "更新后地址", found.Address)
}

func TestParkingLotDAO_Update_ZeroAvailableSpaces(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot := newTestLot("tenant-1", "FullLot")
	lot.TotalSpaces = 50
	lot.AvailableSpaces = 10
	require.NoError(t, dao.Insert(ctx, lot))

	lot.AvailableSpaces = 0
	err := dao.Update(ctx, lot)
	assert.NoError(t, err)

	found, _ := dao.FindByID(ctx, "tenant-1", lot.ID)
	assert.Equal(t, 0, found.AvailableSpaces)
}

func TestParkingLotDAO_Update_NotFound(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	err := dao.Update(ctx, &ParkingLot{ID: "nonexistent", TenantID: "tenant-1"})
	assert.ErrorIs(t, err, errs.ErrParkingLotNotFound)
}

func TestParkingLotDAO_Delete(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot := newTestLot("tenant-1", "朝阳停车场")
	require.NoError(t, dao.Insert(ctx, lot))

	err := dao.Delete(ctx, "tenant-1", lot.ID)
	assert.NoError(t, err)

	_, err = dao.FindByID(ctx, "tenant-1", lot.ID)
	assert.ErrorIs(t, err, errs.ErrParkingLotNotFound)
}

func TestParkingLotDAO_Delete_WrongTenant(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot := newTestLot("tenant-1", "朝阳停车场")
	require.NoError(t, dao.Insert(ctx, lot))

	err := dao.Delete(ctx, "tenant-2", lot.ID)
	assert.ErrorIs(t, err, errs.ErrParkingLotNotFound)
}

func TestParkingLotDAO_Delete_NotFound(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	err := dao.Delete(ctx, "tenant-1", "nonexistent")
	assert.ErrorIs(t, err, errs.ErrParkingLotNotFound)
}

func TestParkingLotDAO_SumStats(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	lot1 := newTestLot("tenant-1", "Lot1")
	lot1.TotalSpaces = 200
	lot1.AvailableSpaces = 150
	require.NoError(t, dao.Insert(ctx, lot1))

	lot2 := newTestLot("tenant-1", "Lot2")
	lot2.TotalSpaces = 100
	lot2.AvailableSpaces = 80
	require.NoError(t, dao.Insert(ctx, lot2))

	require.NoError(t, dao.Insert(ctx, newTestLot("tenant-2", "Other")))

	totalSpaces, availableSpaces, err := dao.SumStats(ctx, "tenant-1")
	assert.NoError(t, err)
	assert.Equal(t, int64(300), totalSpaces)
	assert.Equal(t, int64(230), availableSpaces)
}

func TestParkingLotDAO_SumStats_Empty(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	totalSpaces, availableSpaces, err := dao.SumStats(ctx, "tenant-1")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), totalSpaces)
	assert.Equal(t, int64(0), availableSpaces)
}

func TestParkingLotDAO_Count(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestLot("tenant-1", "Lot1")))
	require.NoError(t, dao.Insert(ctx, newTestLot("tenant-1", "Lot2")))
	require.NoError(t, dao.Insert(ctx, newTestLot("tenant-2", "Other")))

	count, err := dao.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestParkingLotDAO_Count_Empty(t *testing.T) {
	db := setupParkingTestDB(t)
	dao := NewParkingLotDAO(db)
	ctx := context.Background()

	count, err := dao.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
