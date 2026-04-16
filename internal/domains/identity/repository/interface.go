package repository

import (
	"context"

	"github.com/parkhub/api/internal/domains/identity/domain"
)

type TenantFilter struct {
	Status  domain.TenantStatus
	Keyword string
}

//go:generate mockgen -source=./interface.go -package=repomocks -destination=./mocks/repo.mock.go TenantRepo

type TenantRepo interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)
	GetByName(ctx context.Context, name string) (*domain.Tenant, error)
	List(ctx context.Context, filter TenantFilter, page, pageSize int) ([]*domain.Tenant, int64, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	Delete(ctx context.Context, id string) error
	CountByStatus(ctx context.Context) (active, frozen int64, err error)
}
