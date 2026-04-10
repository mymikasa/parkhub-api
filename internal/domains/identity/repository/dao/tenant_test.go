package dao

import (
	"context"
	"testing"

	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Tenant{}))
	return db
}

func newTestTenant(name string) *Tenant {
	return &Tenant{
		ID:           name + "-id",
		Name:         name,
		ContactName:  "Contact " + name,
		ContactPhone: "13800138000",
		ContactEmail: name + "@test.com",
		Status:       "active",
		Address:      "123 Test St",
		PlanType:     "free",
	}
}

func TestTenantDAO_Insert(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	tenant := newTestTenant("Acme")
	err := dao.Insert(ctx, tenant)
	assert.NoError(t, err)
	assert.NotEmpty(t, tenant.ID)
}

func TestTenantDAO_Insert_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	tenant := newTestTenant("Acme")
	require.NoError(t, dao.Insert(ctx, tenant))

	err := dao.Insert(ctx, &Tenant{
		ID:       "dup-id",
		Name:     "Acme",
		Status:   "active",
		PlanType: "free",
	})
	assert.ErrorIs(t, err, errs.ErrTenantAlreadyExists)
}

func TestTenantDAO_FindByID(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	tenant := newTestTenant("Acme")
	require.NoError(t, dao.Insert(ctx, tenant))

	found, err := dao.FindByID(ctx, tenant.ID)
	assert.NoError(t, err)
	assert.Equal(t, tenant.Name, found.Name)
	assert.Equal(t, tenant.ContactEmail, found.ContactEmail)
}

func TestTenantDAO_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	_, err := dao.FindByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, errs.ErrTenantNotFound)
}

func TestTenantDAO_FindByName(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	tenant := newTestTenant("Acme")
	require.NoError(t, dao.Insert(ctx, tenant))

	found, err := dao.FindByName(ctx, "Acme")
	assert.NoError(t, err)
	assert.Equal(t, tenant.ID, found.ID)
}

func TestTenantDAO_FindByName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	_, err := dao.FindByName(ctx, "NonExistent")
	assert.ErrorIs(t, err, errs.ErrTenantNotFound)
}

func TestTenantDAO_FindAll(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		require.NoError(t, dao.Insert(ctx, newTestTenant(name)))
	}

	tenants, total, err := dao.FindAll(ctx, TenantFilter{}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, tenants, 3)
}

func TestTenantDAO_FindAll_WithStatusFilter(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	t1 := newTestTenant("Active")
	t1.Status = "active"
	require.NoError(t, dao.Insert(ctx, t1))

	t2 := newTestTenant("Suspended")
	t2.Status = "frozen"
	require.NoError(t, dao.Insert(ctx, t2))

	tenants, total, err := dao.FindAll(ctx, TenantFilter{Status: "active"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, tenants, 1)
	assert.Equal(t, "active", tenants[0].Status)
}

func TestTenantDAO_FindAll_WithKeywordFilter(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestTenant("Alpha")))
	require.NoError(t, dao.Insert(ctx, newTestTenant("Beta")))

	tenants, total, err := dao.FindAll(ctx, TenantFilter{Keyword: "Alpha"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, tenants, 1)
	assert.Equal(t, "Alpha", tenants[0].Name)
}

func TestTenantDAO_FindAll_Pagination(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, dao.Insert(ctx, newTestTenant(string(rune('A'+i)))))
	}

	tenants, total, err := dao.FindAll(ctx, TenantFilter{}, 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, tenants, 2)
}

func TestTenantDAO_FindAll_EmptyResult(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	tenants, total, err := dao.FindAll(ctx, TenantFilter{}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, tenants)
}

func TestTenantDAO_Update(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	tenant := newTestTenant("Acme")
	require.NoError(t, dao.Insert(ctx, tenant))

	tenant.ContactName = "Updated Contact"
	err := dao.Update(ctx, tenant)
	assert.NoError(t, err)

	found, _ := dao.FindByID(ctx, tenant.ID)
	assert.Equal(t, "Updated Contact", found.ContactName)
}

func TestTenantDAO_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	tenant := &Tenant{ID: "nonexistent", ContactName: "Test"}
	err := dao.Update(ctx, tenant)
	assert.ErrorIs(t, err, errs.ErrTenantNotFound)
}

func TestTenantDAO_Delete(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	tenant := newTestTenant("Acme")
	require.NoError(t, dao.Insert(ctx, tenant))

	err := dao.Delete(ctx, tenant.ID)
	assert.NoError(t, err)

	_, err = dao.FindByID(ctx, tenant.ID)
	assert.ErrorIs(t, err, errs.ErrTenantNotFound)
}

func TestTenantDAO_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	dao := NewTenantDAO(db)
	ctx := context.Background()

	err := dao.Delete(ctx, "nonexistent")
	assert.ErrorIs(t, err, errs.ErrTenantNotFound)
}
