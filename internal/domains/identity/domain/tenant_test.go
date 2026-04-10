package domain

import (
	"testing"

	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/stretchr/testify/assert"
)

func TestNewTenant(t *testing.T) {
	tenant := NewTenant("Acme Corp", "John", "13800138000", "john@acme.com", "123 Main St", PlanTypeFree)

	assert.Equal(t, "Acme Corp", tenant.Name)
	assert.Equal(t, "John", tenant.ContactName)
	assert.Equal(t, "13800138000", tenant.ContactPhone)
	assert.Equal(t, "john@acme.com", tenant.ContactEmail)
	assert.Equal(t, "123 Main St", tenant.Address)
	assert.Equal(t, TenantStatusActive, tenant.Status)
	assert.Equal(t, PlanTypeFree, tenant.PlanType)
}

func TestNewTenant_DifferentPlanTypes(t *testing.T) {
	tests := []struct {
		name     string
		planType PlanType
	}{
		{"free plan", PlanTypeFree},
		{"basic plan", PlanTypeBasic},
		{"pro plan", PlanTypePro},
		{"enterprise plan", PlanTypeEnterprise},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := NewTenant("Corp", "Jane", "13900139000", "jane@corp.com", "", tt.planType)
			assert.Equal(t, tt.planType, tenant.PlanType)
		})
	}
}

func TestNewTenant_EmptyStrings(t *testing.T) {
	tenant := NewTenant("", "", "", "", "", PlanTypeFree)
	assert.Equal(t, "", tenant.Name)
	assert.Equal(t, TenantStatusActive, tenant.Status)
}

func TestTenant_IsActive(t *testing.T) {
	tenant := NewTenant("Corp", "Jane", "13900139000", "jane@corp.com", "", PlanTypeFree)
	assert.True(t, tenant.IsActive())
}

func TestTenant_Freeze(t *testing.T) {
	tenant := NewTenant("Corp", "Jane", "13900139000", "jane@corp.com", "", PlanTypeFree)
	err := tenant.Freeze()
	assert.NoError(t, err)
	assert.Equal(t, TenantStatusFrozen, tenant.Status)
	assert.False(t, tenant.IsActive())
}

func TestTenant_Freeze_AlreadyFrozen(t *testing.T) {
	tenant := NewTenant("Corp", "Jane", "13900139000", "jane@corp.com", "", PlanTypeFree)
	_ = tenant.Freeze()
	err := tenant.Freeze()
	assert.ErrorIs(t, err, errs.ErrTenantInvalidStatus)
}

func TestTenant_Unfreeze(t *testing.T) {
	tenant := NewTenant("Corp", "Jane", "13900139000", "jane@corp.com", "", PlanTypeFree)
	_ = tenant.Freeze()
	err := tenant.Unfreeze()
	assert.NoError(t, err)
	assert.Equal(t, TenantStatusActive, tenant.Status)
	assert.True(t, tenant.IsActive())
}

func TestTenant_Unfreeze_AlreadyActive(t *testing.T) {
	tenant := NewTenant("Corp", "Jane", "13900139000", "jane@corp.com", "", PlanTypeFree)
	err := tenant.Unfreeze()
	assert.ErrorIs(t, err, errs.ErrTenantInvalidStatus)
}

func TestTenant_StatusTransitions(t *testing.T) {
	tenant := NewTenant("Corp", "Jane", "13900139000", "jane@corp.com", "", PlanTypeFree)

	assert.NoError(t, tenant.Freeze())
	assert.Equal(t, TenantStatusFrozen, tenant.Status)

	assert.NoError(t, tenant.Unfreeze())
	assert.Equal(t, TenantStatusActive, tenant.Status)
}
