package domain

import (
	"time"

	"github.com/parkhub/api/services/identity/internal/errs"
)

type UserRole string

const (
	RolePlatformAdmin UserRole = "platform_admin"
	RoleTenantAdmin   UserRole = "tenant_admin"
	RoleOperator      UserRole = "operator"
)

type UserStatus string

const (
	UserStatusActive UserStatus = "active"
	UserStatusFrozen UserStatus = "frozen"
)

type User struct {
	ID           string
	TenantID     *string
	Username     string
	Email        *string
	Phone        *string
	PasswordHash string
	RealName     string
	Role         UserRole
	Status       UserStatus
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(username, passwordHash, realName string, role UserRole, tenantID *string) *User {
	return &User{
		Username:     username,
		PasswordHash: passwordHash,
		RealName:     realName,
		Role:         role,
		TenantID:     tenantID,
		Status:       UserStatusActive,
	}
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

func (u *User) IsPlatformAdmin() bool {
	return u.Role == RolePlatformAdmin
}

func (u *User) Freeze() error {
	if u.Status != UserStatusActive {
		return errs.ErrUserInvalidStatus
	}
	u.Status = UserStatusFrozen
	return nil
}

func (u *User) Unfreeze() error {
	if u.Status != UserStatusFrozen {
		return errs.ErrUserInvalidStatus
	}
	u.Status = UserStatusActive
	return nil
}
