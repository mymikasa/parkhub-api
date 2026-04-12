package domain

import (
	"testing"

	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	tenantID := "tenant-1"
	user := NewUser("alice", "hash123", "Alice", RoleTenantAdmin, &tenantID)

	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, "hash123", user.PasswordHash)
	assert.Equal(t, "Alice", user.RealName)
	assert.Equal(t, RoleTenantAdmin, user.Role)
	assert.Equal(t, &tenantID, user.TenantID)
	assert.Equal(t, UserStatusActive, user.Status)
}

func TestNewUser_PlatformAdmin_NilTenantID(t *testing.T) {
	user := NewUser("admin", "hash", "Admin", RolePlatformAdmin, nil)

	assert.Nil(t, user.TenantID)
	assert.Equal(t, RolePlatformAdmin, user.Role)
}

func TestUser_IsActive(t *testing.T) {
	user := &User{Status: UserStatusActive}
	assert.True(t, user.IsActive())

	user.Status = UserStatusFrozen
	assert.False(t, user.IsActive())
}

func TestUser_IsPlatformAdmin(t *testing.T) {
	user := &User{Role: RolePlatformAdmin}
	assert.True(t, user.IsPlatformAdmin())

	user.Role = RoleTenantAdmin
	assert.False(t, user.IsPlatformAdmin())
}

func TestUser_Freeze_Success(t *testing.T) {
	user := &User{Status: UserStatusActive}
	err := user.Freeze()
	require.NoError(t, err)
	assert.Equal(t, UserStatusFrozen, user.Status)
}

func TestUser_Freeze_AlreadyFrozen(t *testing.T) {
	user := &User{Status: UserStatusFrozen}
	err := user.Freeze()
	assert.ErrorIs(t, err, errs.ErrUserInvalidStatus)
}

func TestUser_Unfreeze_Success(t *testing.T) {
	user := &User{Status: UserStatusFrozen}
	err := user.Unfreeze()
	require.NoError(t, err)
	assert.Equal(t, UserStatusActive, user.Status)
}

func TestUser_Unfreeze_AlreadyActive(t *testing.T) {
	user := &User{Status: UserStatusActive}
	err := user.Unfreeze()
	assert.ErrorIs(t, err, errs.ErrUserInvalidStatus)
}
