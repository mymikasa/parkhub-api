package grpc

import (
	"github.com/parkhub/api/internal/domains/identity/repository"
	"github.com/parkhub/api/internal/domains/identity/repository/dao"
	"github.com/parkhub/api/internal/domains/identity/service"
	identityv1 "github.com/parkhub/api/internal/gen/api/proto/identity/v1"
	"github.com/parkhub/api/internal/registry"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func RegisterServices(reg *registry.Registry, coreDB *gorm.DB) {
	// ── Tenant Service ──
	tenantDAO := dao.NewTenantDAO(coreDB)
	tenantRepo := repository.NewTenantRepo(tenantDAO)
	tenantSvc := service.NewTenantService(tenantRepo)

	reg.MustRegister("identity.v1.TenantService", func(s *grpc.Server) {
		identityv1.RegisterTenantServiceServer(s, NewTenantGRPCServer(tenantSvc))
	})

	// ── User Service ──
	userDAO := dao.NewUserDAO(coreDB)
	userRepo := repository.NewUserRepo(userDAO)
	userSvc := service.NewUserService(userRepo)

	reg.MustRegister("identity.v1.UserService", func(s *grpc.Server) {
		identityv1.RegisterUserServiceServer(s, NewUserGRPCServer(userSvc))
	})
}
