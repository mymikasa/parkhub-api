package grpc

import (
	"context"

	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/service"
	commonv1 "github.com/parkhub/api/internal/gen/api/proto/common/v1"
	identityv1 "github.com/parkhub/api/internal/gen/api/proto/identity/v1"
	"github.com/parkhub/api/pkg/grpcutil"
	"google.golang.org/grpc/codes"
)

type TenantGRPCServer struct {
	identityv1.UnimplementedTenantServiceServer
	tenantSvc service.TenantService
}

func NewTenantGRPCServer(svc service.TenantService) *TenantGRPCServer {
	return &TenantGRPCServer{tenantSvc: svc}
}

var errorMappings = []grpcutil.ErrorMapping{
	{Target: errs.ErrTenantNotFound, Code: codes.NotFound},
	{Target: errs.ErrTenantAlreadyExists, Code: codes.AlreadyExists},
	{Target: errs.ErrTenantInvalidStatus, Code: codes.InvalidArgument},
}

func toGRPCError(err error) error {
	return grpcutil.ToGRPCError(err, errorMappings)
}

func (s *TenantGRPCServer) CreateTenant(ctx context.Context, req *identityv1.CreateTenantRequest) (*identityv1.CreateTenantResponse, error) {
	tenant, err := s.tenantSvc.CreateTenant(ctx, &service.CreateTenantRequest{
		Name:         req.Name,
		ContactName:  req.ContactName,
		ContactPhone: req.ContactPhone,
		ContactEmail: req.ContactEmail,
		Address:      req.Address,
		PlanType:     domainPlanFromProto(req.PlanType),
		Description:  req.GetDescription(),
		CreditCode:   req.GetCreditCode(),
		Remark:       req.GetRemark(),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &identityv1.CreateTenantResponse{Tenant: toProtoTenant(tenant)}, nil
}

func (s *TenantGRPCServer) GetTenant(ctx context.Context, req *identityv1.GetTenantRequest) (*identityv1.GetTenantResponse, error) {
	tenant, err := s.tenantSvc.GetTenant(ctx, req.TenantId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &identityv1.GetTenantResponse{Tenant: toProtoTenant(tenant)}, nil
}

func (s *TenantGRPCServer) ListTenants(ctx context.Context, req *identityv1.ListTenantsRequest) (*identityv1.ListTenantsResponse, error) {
	page := int32(1)
	pageSize := int32(20)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PageSize > 0 {
			pageSize = req.Pagination.PageSize
		}
	}

	resp, err := s.tenantSvc.ListTenants(ctx, &service.ListTenantsRequest{
		Status:   domainStatusFromProto(req.Status),
		Keyword:  req.Keyword,
		Page:     int(page),
		PageSize: int(pageSize),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	tenants := make([]*identityv1.Tenant, 0, len(resp.Tenants))
	for _, t := range resp.Tenants {
		tenants = append(tenants, toProtoTenant(t))
	}

	return &identityv1.ListTenantsResponse{
		Tenants: tenants,
		Pagination: &commonv1.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      resp.Total,
			TotalPages: int32(resp.TotalPages),
		},
	}, nil
}

func (s *TenantGRPCServer) UpdateTenant(ctx context.Context, req *identityv1.UpdateTenantRequest) (*identityv1.UpdateTenantResponse, error) {
	svcReq := &service.UpdateTenantRequest{
		ID: req.TenantId,
	}
	if req.Name != nil {
		svcReq.Name = req.Name
	}
	if req.ContactName != nil {
		svcReq.ContactName = req.ContactName
	}
	if req.ContactPhone != nil {
		svcReq.ContactPhone = req.ContactPhone
	}
	if req.ContactEmail != nil {
		svcReq.ContactEmail = req.ContactEmail
	}
	if req.Address != nil {
		svcReq.Address = req.Address
	}
	if req.PlanType != nil {
		pt := domainPlanFromProto(*req.PlanType)
		svcReq.PlanType = &pt
	}
	if req.Description != nil {
		svcReq.Description = req.Description
	}
	if req.CreditCode != nil {
		svcReq.CreditCode = req.CreditCode
	}
	if req.Remark != nil {
		svcReq.Remark = req.Remark
	}

	tenant, err := s.tenantSvc.UpdateTenant(ctx, svcReq)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &identityv1.UpdateTenantResponse{Tenant: toProtoTenant(tenant)}, nil
}

func (s *TenantGRPCServer) FreezeTenant(ctx context.Context, req *identityv1.FreezeTenantRequest) (*identityv1.FreezeTenantResponse, error) {
	tenant, err := s.tenantSvc.FreezeTenant(ctx, req.TenantId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &identityv1.FreezeTenantResponse{Tenant: toProtoTenant(tenant)}, nil
}

func (s *TenantGRPCServer) UnfreezeTenant(ctx context.Context, req *identityv1.UnfreezeTenantRequest) (*identityv1.UnfreezeTenantResponse, error) {
	tenant, err := s.tenantSvc.UnfreezeTenant(ctx, req.TenantId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &identityv1.UnfreezeTenantResponse{Tenant: toProtoTenant(tenant)}, nil
}

func (s *TenantGRPCServer) DeleteTenant(ctx context.Context, req *identityv1.DeleteTenantRequest) (*identityv1.DeleteTenantResponse, error) {
	if err := s.tenantSvc.DeleteTenant(ctx, req.TenantId); err != nil {
		return nil, toGRPCError(err)
	}
	return &identityv1.DeleteTenantResponse{}, nil
}

func (s *TenantGRPCServer) GetTenantSummary(ctx context.Context, req *identityv1.GetTenantSummaryRequest) (*identityv1.GetTenantSummaryResponse, error) {
	summary, err := s.tenantSvc.GetTenantSummary(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &identityv1.GetTenantSummaryResponse{
		Total:                   summary.Total,
		Active:                  summary.Active,
		Frozen:                  summary.Frozen,
		TotalParkingLots:        summary.TotalParkingLots,
		AvgParkingLotsPerTenant: summary.AvgParkingLotsPerTenant,
	}, nil
}
