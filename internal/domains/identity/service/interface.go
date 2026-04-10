package service

import (
	"context"

	"github.com/parkhub/api/internal/domains/identity/domain"
)

type CreateTenantRequest struct {
	Name         string
	ContactName  string
	ContactPhone string
	ContactEmail string
	Address      string
	PlanType     domain.PlanType
}

type UpdateTenantRequest struct {
	ID           string
	Name         *string
	ContactName  *string
	ContactPhone *string
	ContactEmail *string
	Address      *string
	PlanType     *domain.PlanType
	Status       *domain.TenantStatus
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

//go:generate mockgen -source=./interface.go -package=servicemocks -destination=./mocks/tenant_service.mock.go TenantService

type TenantService interface {
	CreateTenant(ctx context.Context, req *CreateTenantRequest) (*domain.Tenant, error)
	GetTenant(ctx context.Context, id string) (*domain.Tenant, error)
	ListTenants(ctx context.Context, req *ListTenantsRequest) (*TenantListResponse, error)
	UpdateTenant(ctx context.Context, req *UpdateTenantRequest) (*domain.Tenant, error)
	DeleteTenant(ctx context.Context, id string) error
}
