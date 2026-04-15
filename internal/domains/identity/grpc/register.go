package grpc

import (
	"github.com/parkhub/api/internal/config"
	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/repository"
	"github.com/parkhub/api/internal/domains/identity/repository/cache"
	"github.com/parkhub/api/internal/domains/identity/repository/dao"
	"github.com/parkhub/api/internal/domains/identity/service"
	identityv1 "github.com/parkhub/api/internal/gen/api/proto/identity/v1"
	"github.com/parkhub/api/internal/registry"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func RegisterServices(reg *registry.Registry, coreDB *gorm.DB, rdb *redis.Client, authCfg config.AuthConfig) {
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

	// ── Auth Service ──
	signer, err := domain.LoadRS256Signer(authCfg.PrivateKeyPath, authCfg.PublicKeyPath, authCfg.KeyID, authCfg.Issuer)
	if err != nil {
		panic("failed to load RSA signer: " + err.Error())
	}
	refreshTokenRepo := cache.NewRedisRefreshTokenRepo(rdb)
	authSvc := service.NewAuthService(userRepo, refreshTokenRepo, signer, authCfg)

	reg.MustRegister("identity.v1.AuthService", func(s *grpc.Server) {
		identityv1.RegisterAuthServiceServer(s, NewAuthGRPCServer(authSvc, signer))
	})
}
