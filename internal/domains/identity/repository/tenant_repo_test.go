package repository

import (
	"context"
	"testing"

	go_mock "github.com/golang/mock/gomock"
	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/repository/dao"
	daomocks "github.com/parkhub/api/internal/domains/identity/repository/dao/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDAOTenant() *dao.Tenant {
	return &dao.Tenant{
		ID: "test-id", Name: "Acme", ContactName: "John",
		ContactPhone: "13800138000", ContactEmail: "john@acme.com",
		Status: "active", Address: "123 St", PlanType: "free",
	}
}

func TestTenantRepo_Create(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockDAO := daomocks.NewMockTenantDAO(ctrl)
	repo := NewTenantRepo(mockDAO)

	tenant := &domain.Tenant{
		ID:       "test-id",
		Name:     "Acme",
		Status:   domain.TenantStatusActive,
		PlanType: domain.PlanTypeFree,
	}

	mockDAO.EXPECT().Insert(go_mock.Any(), go_mock.Any()).Return(nil)

	err := repo.Create(context.Background(), tenant)
	assert.NoError(t, err)
}

func TestTenantRepo_Create_DAOError(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockDAO := daomocks.NewMockTenantDAO(ctrl)
	repo := NewTenantRepo(mockDAO)

	mockDAO.EXPECT().Insert(go_mock.Any(), go_mock.Any()).Return(errs.ErrTenantAlreadyExists)

	err := repo.Create(context.Background(), &domain.Tenant{Name: "dup"})
	assert.ErrorIs(t, err, errs.ErrTenantAlreadyExists)
}

func TestTenantRepo_GetByID(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockDAO := daomocks.NewMockTenantDAO(ctrl)
	repo := NewTenantRepo(mockDAO)

	mockDAO.EXPECT().FindByID(go_mock.Any(), "test-id").Return(newDAOTenant(), nil)

	tenant, err := repo.GetByID(context.Background(), "test-id")
	assert.NoError(t, err)
	assert.Equal(t, "Acme", tenant.Name)
	assert.Equal(t, domain.TenantStatusActive, tenant.Status)
}

func TestTenantRepo_GetByID_NotFound(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockDAO := daomocks.NewMockTenantDAO(ctrl)
	repo := NewTenantRepo(mockDAO)

	mockDAO.EXPECT().FindByID(go_mock.Any(), "bad-id").Return(nil, errs.ErrTenantNotFound)

	_, err := repo.GetByID(context.Background(), "bad-id")
	assert.ErrorIs(t, err, errs.ErrTenantNotFound)
}

func TestTenantRepo_GetByName(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockDAO := daomocks.NewMockTenantDAO(ctrl)
	repo := NewTenantRepo(mockDAO)

	mockDAO.EXPECT().FindByName(go_mock.Any(), "Acme").Return(newDAOTenant(), nil)

	tenant, err := repo.GetByName(context.Background(), "Acme")
	assert.NoError(t, err)
	assert.Equal(t, "test-id", tenant.ID)
}

func TestTenantRepo_List(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockDAO := daomocks.NewMockTenantDAO(ctrl)
	repo := NewTenantRepo(mockDAO)

	mockDAO.EXPECT().FindAll(go_mock.Any(), go_mock.Any(), 1, 20).Return([]*dao.Tenant{}, int64(0), nil)

	tenants, total, err := repo.List(context.Background(), TenantFilter{}, 1, 20)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, tenants)
}

func TestTenantRepo_Update(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockDAO := daomocks.NewMockTenantDAO(ctrl)
	repo := NewTenantRepo(mockDAO)

	tenant := &domain.Tenant{ID: "test-id", Name: "Updated"}

	mockDAO.EXPECT().Update(go_mock.Any(), go_mock.Any()).Return(nil)

	err := repo.Update(context.Background(), tenant)
	assert.NoError(t, err)
}

func TestTenantRepo_Delete(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockDAO := daomocks.NewMockTenantDAO(ctrl)
	repo := NewTenantRepo(mockDAO)

	mockDAO.EXPECT().Delete(go_mock.Any(), "test-id").Return(nil)

	err := repo.Delete(context.Background(), "test-id")
	assert.NoError(t, err)
}

func TestToDomain_Nil(t *testing.T) {
	result := toDomain(nil)
	assert.Nil(t, result)
}

func TestToEntity_Nil(t *testing.T) {
	result := toEntity(nil)
	assert.Nil(t, result)
}

func TestToDomain_Conversion(t *testing.T) {
	d := newDAOTenant()
	result := toDomain(d)
	require.NotNil(t, result)
	assert.Equal(t, d.ID, result.ID)
	assert.Equal(t, d.Name, result.Name)
	assert.Equal(t, domain.TenantStatus(d.Status), result.Status)
	assert.Equal(t, domain.PlanType(d.PlanType), result.PlanType)
}

func TestToEntity_Conversion(t *testing.T) {
	tenant := &domain.Tenant{
		ID:       "test-id",
		Name:     "Acme",
		Status:   domain.TenantStatusActive,
		PlanType: domain.PlanTypePro,
	}
	result := toEntity(tenant)
	require.NotNil(t, result)
	assert.Equal(t, tenant.ID, result.ID)
	assert.Equal(t, tenant.Name, result.Name)
	assert.Equal(t, string(tenant.Status), result.Status)
	assert.Equal(t, string(tenant.PlanType), result.PlanType)
}
