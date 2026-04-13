package dao

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSmsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&SmsRecord{}))
	return db
}

func strPtr(s string) *string { return &s }
func intPtr(v int64) *int64   { return &v }

func newTestRecord(phone, purpose, status string) *SmsRecord {
	return &SmsRecord{
		ID:      phone + "-" + purpose + "-id",
		Phone:   phone,
		Purpose: purpose,
		Code:    "123456",
		Status:  status,
	}
}

func TestSmsRecordDAO_Insert(t *testing.T) {
	db := setupSmsTestDB(t)
	dao := NewSmsRecordDAO(db)
	ctx := context.Background()

	record := newTestRecord("13800138000", "login", "success")
	err := dao.Insert(ctx, record)
	assert.NoError(t, err)
}

func TestSmsRecordDAO_FindByID(t *testing.T) {
	db := setupSmsTestDB(t)
	dao := NewSmsRecordDAO(db)
	ctx := context.Background()

	record := newTestRecord("13800138000", "login", "success")
	require.NoError(t, dao.Insert(ctx, record))

	found, err := dao.FindByID(ctx, record.ID)
	assert.NoError(t, err)
	assert.Equal(t, record.Phone, found.Phone)
	assert.Equal(t, record.Purpose, found.Purpose)
	assert.Equal(t, record.Code, found.Code)
}

func TestSmsRecordDAO_FindByID_NotFound(t *testing.T) {
	db := setupSmsTestDB(t)
	dao := NewSmsRecordDAO(db)
	ctx := context.Background()

	_, err := dao.FindByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestSmsRecordDAO_FindByPhone(t *testing.T) {
	db := setupSmsTestDB(t)
	dao := NewSmsRecordDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestRecord("13800138000", "login", "success")))
	require.NoError(t, dao.Insert(ctx, newTestRecord("13800138000", "register", "success")))
	require.NoError(t, dao.Insert(ctx, newTestRecord("13900139000", "login", "success")))

	records, total, err := dao.FindByPhone(ctx, "13800138000", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, records, 2)
}

func TestSmsRecordDAO_FindByPhone_Pagination(t *testing.T) {
	db := setupSmsTestDB(t)
	dao := NewSmsRecordDAO(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		r := newTestRecord("13800138000", "login", "success")
		r.ID = r.ID + string(rune('a'+i))
		require.NoError(t, dao.Insert(ctx, r))
	}

	records, total, err := dao.FindByPhone(ctx, "13800138000", 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, records, 2)
}

func TestSmsRecordDAO_List(t *testing.T) {
	db := setupSmsTestDB(t)
	dao := NewSmsRecordDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestRecord("13800138000", "login", "success")))
	require.NoError(t, dao.Insert(ctx, newTestRecord("13900139000", "register", "success")))

	records, total, err := dao.List(ctx, SmsRecordFilter{}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, records, 2)
}

func TestSmsRecordDAO_List_WithPhoneFilter(t *testing.T) {
	db := setupSmsTestDB(t)
	dao := NewSmsRecordDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestRecord("13800138000", "login", "success")))
	require.NoError(t, dao.Insert(ctx, newTestRecord("13900139000", "login", "success")))

	records, total, err := dao.List(ctx, SmsRecordFilter{Phone: "13800138000"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, records, 1)
	assert.Equal(t, "13800138000", records[0].Phone)
}

func TestSmsRecordDAO_List_WithStatusFilter(t *testing.T) {
	db := setupSmsTestDB(t)
	dao := NewSmsRecordDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestRecord("13800138000", "login", "success")))

	failed := newTestRecord("13800138000", "register", "failed")
	failed.ID = "failed-id"
	failed.ProviderErr = strPtr("timeout")
	require.NoError(t, dao.Insert(ctx, failed))

	records, total, err := dao.List(ctx, SmsRecordFilter{Status: "failed"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, records, 1)
	assert.Equal(t, "failed", records[0].Status)
}

func TestSmsRecordDAO_Insert_WithProviderErr(t *testing.T) {
	db := setupSmsTestDB(t)
	dao := NewSmsRecordDAO(db)
	ctx := context.Background()

	record := newTestRecord("13800138000", "login", "failed")
	record.ProviderErr = strPtr("gateway timeout")
	require.NoError(t, dao.Insert(ctx, record))

	found, err := dao.FindByID(ctx, record.ID)
	require.NoError(t, err)
	assert.Equal(t, "gateway timeout", *found.ProviderErr)
}
