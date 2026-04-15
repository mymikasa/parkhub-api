package grpc

import (
	"github.com/parkhub/api/internal/domains/identity/domain"
	identityv1 "github.com/parkhub/api/internal/gen/api/proto/identity/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoTenant(t *domain.Tenant) *identityv1.Tenant {
	if t == nil {
		return nil
	}
	return &identityv1.Tenant{
		TenantId:        t.ID,
		Name:            t.Name,
		ContactName:     t.ContactName,
		ContactPhone:    t.ContactPhone,
		ContactEmail:    t.ContactEmail,
		Status:          domainStatusToProto(t.Status),
		Address:         t.Address,
		PlanType:        domainPlanToProto(t.PlanType),
		CreatedAt:       toTimestamp(t.CreatedAt),
		UpdatedAt:       toTimestamp(t.UpdatedAt),
		Description:     t.Description,
		CreditCode:      t.CreditCode,
		Remark:          t.Remark,
		ParkingLotCount: t.ParkingLotCount,
	}
}

func domainStatusToProto(s domain.TenantStatus) identityv1.TenantStatus {
	switch s {
	case domain.TenantStatusActive:
		return identityv1.TenantStatus_TENANT_STATUS_ACTIVE
	case domain.TenantStatusFrozen:
		return identityv1.TenantStatus_TENANT_STATUS_FROZEN
	default:
		return identityv1.TenantStatus_TENANT_STATUS_UNSPECIFIED
	}
}

func domainStatusFromProto(s identityv1.TenantStatus) domain.TenantStatus {
	switch s {
	case identityv1.TenantStatus_TENANT_STATUS_ACTIVE:
		return domain.TenantStatusActive
	case identityv1.TenantStatus_TENANT_STATUS_FROZEN:
		return domain.TenantStatusFrozen
	default:
		return domain.TenantStatusActive
	}
}

func domainPlanToProto(p domain.PlanType) identityv1.PlanType {
	switch p {
	case domain.PlanTypeFree:
		return identityv1.PlanType_PLAN_TYPE_FREE
	case domain.PlanTypeBasic:
		return identityv1.PlanType_PLAN_TYPE_BASIC
	case domain.PlanTypePro:
		return identityv1.PlanType_PLAN_TYPE_PRO
	case domain.PlanTypeEnterprise:
		return identityv1.PlanType_PLAN_TYPE_ENTERPRISE
	default:
		return identityv1.PlanType_PLAN_TYPE_UNSPECIFIED
	}
}

func domainPlanFromProto(p identityv1.PlanType) domain.PlanType {
	switch p {
	case identityv1.PlanType_PLAN_TYPE_FREE:
		return domain.PlanTypeFree
	case identityv1.PlanType_PLAN_TYPE_BASIC:
		return domain.PlanTypeBasic
	case identityv1.PlanType_PLAN_TYPE_PRO:
		return domain.PlanTypePro
	case identityv1.PlanType_PLAN_TYPE_ENTERPRISE:
		return domain.PlanTypeEnterprise
	default:
		return domain.PlanTypeFree
	}
}

func toTimestamp(millis int64) *timestamppb.Timestamp {
	if millis == 0 {
		return nil
	}
	seconds := millis / 1000
	nanos := int32((millis % 1000) * 1e6)
	return &timestamppb.Timestamp{Seconds: seconds, Nanos: nanos}
}
