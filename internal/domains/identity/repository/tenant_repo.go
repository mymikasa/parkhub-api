package repository

import (
	"context"

	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/repository/dao"
)

type tenantRepo struct {
	dao dao.TenantDAO
}

func NewTenantRepo(d dao.TenantDAO) TenantRepo {
	return &tenantRepo{dao: d}
}

func (r *tenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	return r.dao.Insert(ctx, toEntity(tenant))
}

func (r *tenantRepo) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	d, err := r.dao.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDomain(d), nil
}

func (r *tenantRepo) GetByName(ctx context.Context, name string) (*domain.Tenant, error) {
	d, err := r.dao.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return toDomain(d), nil
}

func (r *tenantRepo) List(ctx context.Context, filter TenantFilter, page, pageSize int) ([]*domain.Tenant, int64, error) {
	daoFilter := dao.TenantFilter{
		Status:  string(filter.Status),
		Keyword: filter.Keyword,
	}
	records, total, err := r.dao.FindAll(ctx, daoFilter, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	tenants := make([]*domain.Tenant, 0, len(records))
	for _, r := range records {
		tenants = append(tenants, toDomain(r))
	}
	return tenants, total, nil
}

func (r *tenantRepo) Update(ctx context.Context, tenant *domain.Tenant) error {
	return r.dao.Update(ctx, toEntity(tenant))
}

func (r *tenantRepo) Delete(ctx context.Context, id string) error {
	return r.dao.Delete(ctx, id)
}

func toDomain(d *dao.Tenant) *domain.Tenant {
	if d == nil {
		return nil
	}
	return &domain.Tenant{
		ID:           d.ID,
		Name:         d.Name,
		ContactName:  d.ContactName,
		ContactPhone: d.ContactPhone,
		ContactEmail: d.ContactEmail,
		Status:       domain.TenantStatus(d.Status),
		Address:      d.Address,
		PlanType:     domain.PlanType(d.PlanType),
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

func toEntity(t *domain.Tenant) *dao.Tenant {
	if t == nil {
		return nil
	}
	return &dao.Tenant{
		ID:           t.ID,
		Name:         t.Name,
		ContactName:  t.ContactName,
		ContactPhone: t.ContactPhone,
		ContactEmail: t.ContactEmail,
		Status:       string(t.Status),
		Address:      t.Address,
		PlanType:     string(t.PlanType),
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}
