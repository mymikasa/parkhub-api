package service

import (
	"context"
	"math"

	"github.com/google/uuid"
	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/repository"
)

type tenantService struct {
	tenantRepo     repository.TenantRepo
	parkingCounter ParkingLotCounter
}

func NewTenantService(repo repository.TenantRepo, parkingCounter ParkingLotCounter) TenantService {
	return &tenantService{tenantRepo: repo, parkingCounter: parkingCounter}
}

func (s *tenantService) CreateTenant(ctx context.Context, req *CreateTenantRequest) (*domain.Tenant, error) {
	existing, err := s.tenantRepo.GetByName(ctx, req.Name)
	if err == nil && existing != nil {
		return nil, errs.ErrTenantAlreadyExists
	}

	tenant := domain.NewTenant(
		req.Name,
		req.ContactName,
		req.ContactPhone,
		req.ContactEmail,
		req.Address,
		req.PlanType,
	)
	tenant.ID = uuid.New().String()
	tenant.Description = req.Description
	tenant.CreditCode = req.CreditCode
	tenant.Remark = req.Remark

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *tenantService) GetTenant(ctx context.Context, id string) (*domain.Tenant, error) {
	return s.tenantRepo.GetByID(ctx, id)
}

func (s *tenantService) ListTenants(ctx context.Context, req *ListTenantsRequest) (*TenantListResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filter := repository.TenantFilter{
		Status:  req.Status,
		Keyword: req.Keyword,
	}

	tenants, total, err := s.tenantRepo.List(ctx, filter, page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &TenantListResponse{
		Tenants:    tenants,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *tenantService) UpdateTenant(ctx context.Context, req *UpdateTenantRequest) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.ContactName != nil {
		tenant.ContactName = *req.ContactName
	}
	if req.ContactPhone != nil {
		tenant.ContactPhone = *req.ContactPhone
	}
	if req.ContactEmail != nil {
		tenant.ContactEmail = *req.ContactEmail
	}
	if req.Address != nil {
		tenant.Address = *req.Address
	}
	if req.PlanType != nil {
		tenant.PlanType = *req.PlanType
	}
	if req.Description != nil {
		tenant.Description = *req.Description
	}
	if req.CreditCode != nil {
		tenant.CreditCode = *req.CreditCode
	}
	if req.Remark != nil {
		tenant.Remark = *req.Remark
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *tenantService) FreezeTenant(ctx context.Context, id string) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := tenant.Freeze(); err != nil {
		return nil, err
	}
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *tenantService) UnfreezeTenant(ctx context.Context, id string) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := tenant.Unfreeze(); err != nil {
		return nil, err
	}
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *tenantService) DeleteTenant(ctx context.Context, id string) error {
	return s.tenantRepo.Delete(ctx, id)
}

func (s *tenantService) GetTenantSummary(ctx context.Context) (*TenantSummaryResponse, error) {
	active, frozen, err := s.tenantRepo.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}

	total := active + frozen

	var totalParkingLots int64
	if s.parkingCounter != nil {
		totalParkingLots, _ = s.parkingCounter.CountParkingLots(ctx)
	}

	var avg float64
	if total > 0 {
		avg = float64(totalParkingLots) / float64(total)
	}

	return &TenantSummaryResponse{
		Total:                   total,
		Active:                  active,
		Frozen:                  frozen,
		TotalParkingLots:        totalParkingLots,
		AvgParkingLotsPerTenant: avg,
	}, nil
}
