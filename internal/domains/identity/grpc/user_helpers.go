package grpc

import (
	"github.com/parkhub/api/internal/domains/identity/domain"
	identityv1 "github.com/parkhub/api/internal/gen/api/proto/identity/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProtoUser(u *domain.User) *identityv1.User {
	if u == nil {
		return nil
	}
	pb := &identityv1.User{
		UserId:    u.ID,
		TenantId:  u.TenantID,
		Username:  u.Username,
		Email:     u.Email,
		Phone:     u.Phone,
		RealName:  u.RealName,
		Role:      domainRoleToProto(u.Role),
		Status:    domainUserStatusToProto(u.Status),
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
	if u.LastLoginAt != nil {
		pb.LastLoginAt = timestamppb.New(*u.LastLoginAt)
	}
	return pb
}

func domainRoleToProto(r domain.UserRole) identityv1.UserRole {
	switch r {
	case domain.RolePlatformAdmin:
		return identityv1.UserRole_USER_ROLE_PLATFORM_ADMIN
	case domain.RoleTenantAdmin:
		return identityv1.UserRole_USER_ROLE_TENANT_ADMIN
	case domain.RoleOperator:
		return identityv1.UserRole_USER_ROLE_OPERATOR
	default:
		return identityv1.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func domainRoleFromProto(r identityv1.UserRole) domain.UserRole {
	switch r {
	case identityv1.UserRole_USER_ROLE_PLATFORM_ADMIN:
		return domain.RolePlatformAdmin
	case identityv1.UserRole_USER_ROLE_TENANT_ADMIN:
		return domain.RoleTenantAdmin
	case identityv1.UserRole_USER_ROLE_OPERATOR:
		return domain.RoleOperator
	default:
		return ""
	}
}

func domainUserStatusToProto(s domain.UserStatus) identityv1.UserStatus {
	switch s {
	case domain.UserStatusActive:
		return identityv1.UserStatus_USER_STATUS_ACTIVE
	case domain.UserStatusFrozen:
		return identityv1.UserStatus_USER_STATUS_FROZEN
	default:
		return identityv1.UserStatus_USER_STATUS_UNSPECIFIED
	}
}

func domainUserStatusFromProto(s identityv1.UserStatus) domain.UserStatus {
	switch s {
	case identityv1.UserStatus_USER_STATUS_ACTIVE:
		return domain.UserStatusActive
	case identityv1.UserStatus_USER_STATUS_FROZEN:
		return domain.UserStatusFrozen
	default:
		return ""
	}
}
