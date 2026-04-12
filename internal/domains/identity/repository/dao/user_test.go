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

func setupUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}))
	return db
}

func strPtr(s string) *string { return &s }

func newTestUser(username string, tenantID *string) *User {
	return &User{
		ID:           username + "-id",
		TenantID:     tenantID,
		Username:     username,
		Email:        strPtr(username + "@test.com"),
		Phone:        strPtr("13800138000"),
		PasswordHash: "hashed",
		RealName:     "Real " + username,
		Role:         "operator",
		Status:       "active",
	}
}

func TestUserDAO_Insert(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := newTestUser("alice", strPtr("tenant-1"))
	err := dao.Insert(ctx, user)
	assert.NoError(t, err)
}

func TestUserDAO_Insert_Duplicate(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestUser("alice", strPtr("tenant-1"))))

	dup := newTestUser("alice", strPtr("tenant-1"))
	dup.ID = "dup-id"
	err := dao.Insert(ctx, dup)
	assert.ErrorIs(t, err, errs.ErrUserAlreadyExists)
}

func TestUserDAO_FindByID(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := newTestUser("alice", strPtr("tenant-1"))
	require.NoError(t, dao.Insert(ctx, user))

	found, err := dao.FindByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, user.Username, found.Username)
	assert.Equal(t, user.TenantID, found.TenantID)
}

func TestUserDAO_FindByID_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	_, err := dao.FindByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, errs.ErrUserNotFound)
}

func TestUserDAO_FindByUsername(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := newTestUser("alice", strPtr("tenant-1"))
	require.NoError(t, dao.Insert(ctx, user))

	found, err := dao.FindByUsername(ctx, "alice")
	assert.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
}

func TestUserDAO_FindByUsername_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	_, err := dao.FindByUsername(ctx, "nonexistent")
	assert.ErrorIs(t, err, errs.ErrUserNotFound)
}

func TestUserDAO_FindAll(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	tid := strPtr("tenant-1")
	for _, name := range []string{"alice", "bob", "charlie"} {
		require.NoError(t, dao.Insert(ctx, newTestUser(name, tid)))
	}

	users, total, err := dao.FindAll(ctx, UserFilter{}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, users, 3)
}

func TestUserDAO_FindAll_WithTenantFilter(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestUser("alice", strPtr("tenant-1"))))
	require.NoError(t, dao.Insert(ctx, newTestUser("bob", strPtr("tenant-2"))))

	users, total, err := dao.FindAll(ctx, UserFilter{TenantID: "tenant-1"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Username)
}

func TestUserDAO_FindAll_WithRoleFilter(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	u1 := newTestUser("admin", nil)
	u1.Role = "platform_admin"
	require.NoError(t, dao.Insert(ctx, u1))

	u2 := newTestUser("op", strPtr("t1"))
	u2.Role = "operator"
	require.NoError(t, dao.Insert(ctx, u2))

	users, total, err := dao.FindAll(ctx, UserFilter{Role: "platform_admin"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
}

func TestUserDAO_FindAll_WithKeyword(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	require.NoError(t, dao.Insert(ctx, newTestUser("alice", strPtr("t1"))))
	require.NoError(t, dao.Insert(ctx, newTestUser("bob", strPtr("t1"))))

	users, total, err := dao.FindAll(ctx, UserFilter{Keyword: "alice"}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
}

func TestUserDAO_FindAll_Pagination(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, dao.Insert(ctx, newTestUser(string(rune('a'+i)), strPtr("t1"))))
	}

	users, total, err := dao.FindAll(ctx, UserFilter{}, 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, users, 2)
}

func TestUserDAO_FindAll_Empty(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	users, total, err := dao.FindAll(ctx, UserFilter{}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, users)
}

func TestUserDAO_Update(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := newTestUser("alice", strPtr("t1"))
	require.NoError(t, dao.Insert(ctx, user))

	user.RealName = "Updated Alice"
	user.Status = "frozen"
	err := dao.Update(ctx, user)
	assert.NoError(t, err)

	found, _ := dao.FindByID(ctx, user.ID)
	assert.Equal(t, "Updated Alice", found.RealName)
	assert.Equal(t, "frozen", found.Status)
}

func TestUserDAO_Update_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &User{ID: "nonexistent", RealName: "Test"}
	err := dao.Update(ctx, user)
	assert.ErrorIs(t, err, errs.ErrUserNotFound)
}

func TestUserDAO_BatchInsert(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	users := []*User{
		newTestUser("alice", strPtr("t1")),
		newTestUser("bob", strPtr("t1")),
	}
	err := dao.BatchInsert(ctx, users)
	assert.NoError(t, err)

	all, total, err := dao.FindAll(ctx, UserFilter{}, 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, all, 2)
}

func TestUserDAO_Insert_NilTenantID(t *testing.T) {
	db := setupUserTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := newTestUser("admin", nil)
	user.Role = "platform_admin"
	err := dao.Insert(ctx, user)
	assert.NoError(t, err)

	found, err := dao.FindByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Nil(t, found.TenantID)
}
