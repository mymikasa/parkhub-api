package service

import (
	"context"

	"github.com/parkhub/api/services/identity/internal/domain"
)

type CreateTenantRequest struct {
	Name         string
	ContactName  string
	ContactPhone string
	ContactEmail string
	Address      string
	PlanType     domain.PlanType
	Description  string
	CreditCode   string
	Remark       string
}

type UpdateTenantRequest struct {
	ID           string
	Name         *string
	ContactName  *string
	ContactPhone *string
	ContactEmail *string
	Address      *string
	PlanType     *domain.PlanType
	Description  *string
	CreditCode   *string
	Remark       *string
}

type ListTenantsRequest struct {
	Status   domain.TenantStatus
	Keyword  string
	Page     int
	PageSize int
}

type TenantListResponse struct {
	Tenants    []*domain.Tenant
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

type TenantSummaryResponse struct {
	Total  int64
	Active int64
	Frozen int64
}

type TenantService interface {
	CreateTenant(ctx context.Context, req *CreateTenantRequest) (*domain.Tenant, error)
	GetTenant(ctx context.Context, id string) (*domain.Tenant, error)
	ListTenants(ctx context.Context, req *ListTenantsRequest) (*TenantListResponse, error)
	UpdateTenant(ctx context.Context, req *UpdateTenantRequest) (*domain.Tenant, error)
	FreezeTenant(ctx context.Context, id string) (*domain.Tenant, error)
	UnfreezeTenant(ctx context.Context, id string) (*domain.Tenant, error)
	DeleteTenant(ctx context.Context, id string) error
	GetTenantSummary(ctx context.Context) (*TenantSummaryResponse, error)
}
