package repository

import (
	"context"
	"testing"

	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/repository/dao"
	daomocks "github.com/parkhub/api/internal/domains/identity/repository/dao/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUserRepo_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDAO := daomocks.NewMockUserDAO(ctrl)

	mockDAO.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)

	repo := NewUserRepo(mockDAO)
	tenantID := "tenant-1"
	err := repo.Create(context.Background(), &domain.User{
		ID:       "user-1",
		TenantID: &tenantID,
		Username: "alice",
		Role:     domain.RoleTenantAdmin,
		Status:   domain.UserStatusActive,
	})
	assert.NoError(t, err)
}

func TestUserRepo_GetByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDAO := daomocks.NewMockUserDAO(ctrl)

	tenantID := "tenant-1"
	mockDAO.EXPECT().FindByID(gomock.Any(), "user-1").Return(&dao.User{
		ID:       "user-1",
		TenantID: &tenantID,
		Username: "alice",
		Role:     "tenant_admin",
		Status:   "active",
	}, nil)

	repo := NewUserRepo(mockDAO)
	user, err := repo.GetByID(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, domain.RoleTenantAdmin, user.Role)
	assert.Equal(t, domain.UserStatusActive, user.Status)
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDAO := daomocks.NewMockUserDAO(ctrl)

	mockDAO.EXPECT().FindByID(gomock.Any(), "nonexistent").Return(nil, errs.ErrUserNotFound)

	repo := NewUserRepo(mockDAO)
	_, err := repo.GetByID(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, errs.ErrUserNotFound)
}

func TestUserRepo_GetByUsername(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDAO := daomocks.NewMockUserDAO(ctrl)

	mockDAO.EXPECT().FindByUsername(gomock.Any(), "alice").Return(&dao.User{
		ID:       "user-1",
		Username: "alice",
		Role:     "operator",
		Status:   "active",
	}, nil)

	repo := NewUserRepo(mockDAO)
	user, err := repo.GetByUsername(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, "user-1", user.ID)
}

func TestUserRepo_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDAO := daomocks.NewMockUserDAO(ctrl)

	mockDAO.EXPECT().FindAll(gomock.Any(), dao.UserFilter{TenantID: "t1"}, 1, 20).
		Return([]*dao.User{
			{ID: "u1", Username: "alice", Role: "operator", Status: "active"},
			{ID: "u2", Username: "bob", Role: "operator", Status: "active"},
		}, int64(2), nil)

	repo := NewUserRepo(mockDAO)
	users, total, err := repo.List(context.Background(), UserFilter{TenantID: "t1"}, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, users, 2)
}

func TestUserRepo_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDAO := daomocks.NewMockUserDAO(ctrl)

	mockDAO.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	repo := NewUserRepo(mockDAO)
	err := repo.Update(context.Background(), &domain.User{
		ID:       "user-1",
		Username: "alice",
		RealName: "Updated",
		Role:     domain.RoleOperator,
		Status:   domain.UserStatusActive,
	})
	assert.NoError(t, err)
}

func TestUserRepo_BatchCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDAO := daomocks.NewMockUserDAO(ctrl)

	mockDAO.EXPECT().BatchInsert(gomock.Any(), gomock.Len(2)).Return(nil)

	repo := NewUserRepo(mockDAO)
	err := repo.BatchCreate(context.Background(), []*domain.User{
		{ID: "u1", Username: "alice"},
		{ID: "u2", Username: "bob"},
	})
	assert.NoError(t, err)
}

func TestToDomainUser_NilLastLoginAt(t *testing.T) {
	d := &dao.User{
		ID:          "u1",
		Username:    "alice",
		Role:        "operator",
		Status:      "active",
		LastLoginAt: nil,
	}
	u := toDomainUser(d)
	assert.Nil(t, u.LastLoginAt)
}

func TestToDomainUser_WithLastLoginAt(t *testing.T) {
	millis := int64(1700000000000)
	d := &dao.User{
		ID:          "u1",
		Username:    "alice",
		Role:        "operator",
		Status:      "active",
		LastLoginAt: &millis,
	}
	u := toDomainUser(d)
	require.NotNil(t, u.LastLoginAt)
	assert.Equal(t, millis, u.LastLoginAt.UnixMilli())
}
