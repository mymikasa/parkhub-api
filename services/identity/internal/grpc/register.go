package grpc

import (
	"github.com/parkhub/api/services/identity/internal/domain"
	identityv1 "github.com/parkhub/api/services/identity/internal/gen/api/proto/identity/v1"
	smsv1 "github.com/parkhub/api/services/identity/internal/gen/api/proto/sms/v1"
	"github.com/parkhub/api/services/identity/internal/registry"
	"github.com/parkhub/api/services/identity/internal/repository"
	"github.com/parkhub/api/services/identity/internal/repository/cache"
	"github.com/parkhub/api/services/identity/internal/repository/dao"
	"github.com/parkhub/api/services/identity/internal/service"
	"github.com/parkhub/api/services/identity/internal/service/client"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type Config struct {
	Issuer         string
	AccessTTL      string
	RefreshTTL     string
	PrivateKeyPath string
	PublicKeyPath  string
	KeyID          string
}

func RegisterServices(reg *registry.Registry, coreDB *gorm.DB, rdb *redis.Client, authCfg Config, smsClient smsv1.SmsServiceClient) {
	tenantDAO := dao.NewTenantDAO(coreDB)
	tenantRepo := repository.NewTenantRepo(tenantDAO)
	parkingCounter := client.NewNoopParkingLotCounter()
	tenantSvc := service.NewTenantService(tenantRepo, parkingCounter)

	reg.MustRegister("identity.v1.TenantService", func(s *grpc.Server) {
		identityv1.RegisterTenantServiceServer(s, NewTenantGRPCServer(tenantSvc))
	})

	userDAO := dao.NewUserDAO(coreDB)
	userRepo := repository.NewUserRepo(userDAO)
	userSvc := service.NewUserService(userRepo)

	reg.MustRegister("identity.v1.UserService", func(s *grpc.Server) {
		identityv1.RegisterUserServiceServer(s, NewUserGRPCServer(userSvc))
	})

	signer, err := domain.LoadRS256Signer(authCfg.PrivateKeyPath, authCfg.PublicKeyPath, authCfg.KeyID, authCfg.Issuer)
	if err != nil {
		panic("failed to load RSA signer: " + err.Error())
	}
	refreshTokenRepo := cache.NewRedisRefreshTokenRepo(rdb)
	smsVerifier := client.NewSmsVerifierClient(smsClient)
	authSvc := service.NewAuthService(userRepo, refreshTokenRepo, signer, service.AuthConfig(authCfg), smsVerifier)

	reg.MustRegister("identity.v1.AuthService", func(s *grpc.Server) {
		identityv1.RegisterAuthServiceServer(s, NewAuthGRPCServer(authSvc, signer))
	})
}
