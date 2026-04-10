package service

import (
	"context"
	"testing"

	go_mock "github.com/golang/mock/gomock"
	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	repomocks "github.com/parkhub/api/internal/domains/identity/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantService_CreateTenant_Success(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().GetByName(go_mock.Any(), "Acme").Return(nil, errs.ErrTenantNotFound)
	mockRepo.EXPECT().Create(go_mock.Any(), go_mock.Any()).Return(nil)

	tenant, err := svc.CreateTenant(context.Background(), &CreateTenantRequest{
		Name:         "Acme",
		ContactName:  "John",
		ContactPhone: "13800138000",
		ContactEmail: "john@acme.com",
		Address:      "123 St",
		PlanType:     domain.PlanTypeFree,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, tenant.ID)
	assert.Equal(t, "Acme", tenant.Name)
	assert.Equal(t, domain.TenantStatusActive, tenant.Status)
}

func TestTenantService_CreateTenant_AlreadyExists(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().GetByName(go_mock.Any(), "Acme").Return(&domain.Tenant{ID: "existing-id", Name: "Acme"}, nil)

	_, err := svc.CreateTenant(context.Background(), &CreateTenantRequest{
		Name: "Acme",
	})
	assert.ErrorIs(t, err, errs.ErrTenantAlreadyExists)
}

func TestTenantService_GetTenant_Success(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().GetByID(go_mock.Any(), "test-id").Return(&domain.Tenant{ID: "test-id", Name: "Acme"}, nil)

	tenant, err := svc.GetTenant(context.Background(), "test-id")
	assert.NoError(t, err)
	assert.Equal(t, "test-id", tenant.ID)
}

func TestTenantService_GetTenant_NotFound(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().GetByID(go_mock.Any(), "bad-id").Return(nil, errs.ErrTenantNotFound)

	_, err := svc.GetTenant(context.Background(), "bad-id")
	assert.ErrorIs(t, err, errs.ErrTenantNotFound)
}

func TestTenantService_ListTenants_Defaults(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().List(go_mock.Any(), go_mock.Any(), 1, 20).Return([]*domain.Tenant{}, int64(0), nil)

	resp, err := svc.ListTenants(context.Background(), &ListTenantsRequest{})
	assert.NoError(t, err)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
	assert.Equal(t, int64(0), resp.Total)
	assert.Equal(t, 0, resp.TotalPages)
}

func TestTenantService_ListTenants_WithPagination(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	tenants := make([]*domain.Tenant, 5)
	for i := range tenants {
		tenants[i] = &domain.Tenant{ID: "id", Name: "name"}
	}
	mockRepo.EXPECT().List(go_mock.Any(), go_mock.Any(), 2, 5).Return(tenants, int64(50), nil)

	resp, err := svc.ListTenants(context.Background(), &ListTenantsRequest{Page: 2, PageSize: 5})
	assert.NoError(t, err)
	assert.Equal(t, 2, resp.Page)
	assert.Equal(t, 5, resp.PageSize)
	assert.Equal(t, int64(50), resp.Total)
	assert.Equal(t, 10, resp.TotalPages)
}

func TestTenantService_ListTenants_PageSizeCap(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().List(go_mock.Any(), go_mock.Any(), 1, 100).Return([]*domain.Tenant{}, int64(0), nil)

	resp, err := svc.ListTenants(context.Background(), &ListTenantsRequest{Page: 1, PageSize: 200})
	assert.NoError(t, err)
	assert.Equal(t, 100, resp.PageSize)
}

func TestTenantService_UpdateTenant_Success(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	existing := &domain.Tenant{ID: "test-id", Name: "Acme", Status: domain.TenantStatusActive}
	mockRepo.EXPECT().GetByID(go_mock.Any(), "test-id").Return(existing, nil)
	mockRepo.EXPECT().Update(go_mock.Any(), go_mock.Any()).Return(nil)

	newName := "Updated"
	tenant, err := svc.UpdateTenant(context.Background(), &UpdateTenantRequest{
		ID:   "test-id",
		Name: &newName,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Updated", tenant.Name)
}

func TestTenantService_UpdateTenant_StatusTransition(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	existing := &domain.Tenant{ID: "test-id", Name: "Acme", Status: domain.TenantStatusActive}
	mockRepo.EXPECT().GetByID(go_mock.Any(), "test-id").Return(existing, nil)
	mockRepo.EXPECT().Update(go_mock.Any(), go_mock.Any()).Return(nil)

	frozen := domain.TenantStatusFrozen
	tenant, err := svc.UpdateTenant(context.Background(), &UpdateTenantRequest{
		ID:     "test-id",
		Status: &frozen,
	})
	assert.NoError(t, err)
	assert.Equal(t, domain.TenantStatusFrozen, tenant.Status)
}

func TestTenantService_UpdateTenant_InvalidStatusTransition(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	existing := &domain.Tenant{ID: "test-id", Name: "Acme", Status: domain.TenantStatusFrozen}
	mockRepo.EXPECT().GetByID(go_mock.Any(), "test-id").Return(existing, nil)

	frozen := domain.TenantStatusFrozen
	_, err := svc.UpdateTenant(context.Background(), &UpdateTenantRequest{
		ID:     "test-id",
		Status: &frozen,
	})
	assert.ErrorIs(t, err, errs.ErrTenantInvalidStatus)
}

func TestTenantService_UpdateTenant_NotFound(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().GetByID(go_mock.Any(), "bad-id").Return(nil, errs.ErrTenantNotFound)

	_, err := svc.UpdateTenant(context.Background(), &UpdateTenantRequest{ID: "bad-id"})
	assert.ErrorIs(t, err, errs.ErrTenantNotFound)
}

func TestTenantService_DeleteTenant_Success(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().Delete(go_mock.Any(), "test-id").Return(nil)

	err := svc.DeleteTenant(context.Background(), "test-id")
	assert.NoError(t, err)
}

func TestTenantService_DeleteTenant_NotFound(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().Delete(go_mock.Any(), "bad-id").Return(errs.ErrTenantNotFound)

	err := svc.DeleteTenant(context.Background(), "bad-id")
	assert.ErrorIs(t, err, errs.ErrTenantNotFound)
}

func TestTenantService_ListTenants_TotalPagesCeiling(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockTenantRepo(ctrl)
	svc := NewTenantService(mockRepo)

	mockRepo.EXPECT().List(go_mock.Any(), go_mock.Any(), 1, 20).Return([]*domain.Tenant{}, int64(25), nil)

	resp, err := svc.ListTenants(context.Background(), &ListTenantsRequest{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, 2, resp.TotalPages)
}
