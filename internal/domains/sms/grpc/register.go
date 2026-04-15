package grpc

import (
	"github.com/parkhub/api/internal/domains/sms/gateway"
	"github.com/parkhub/api/internal/domains/sms/repository"
	"github.com/parkhub/api/internal/domains/sms/repository/cache"
	"github.com/parkhub/api/internal/domains/sms/repository/dao"
	"github.com/parkhub/api/internal/domains/sms/service"
	smsv1 "github.com/parkhub/api/internal/gen/api/proto/sms/v1"
	"github.com/parkhub/api/internal/registry"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func RegisterServices(reg *registry.Registry, coreDB *gorm.DB, rdb *redis.Client) service.SmsService {
	smsDAO := dao.NewSmsRecordDAO(coreDB)
	smsCache := cache.NewRedisSmsCache(rdb)
	smsRepo := repository.NewSmsRepository(smsDAO, smsCache)
	smsGateway := gateway.NewMockSmsGateway()
	smsSvc := service.NewSmsService(smsRepo, smsGateway)

	reg.MustRegister("sms.v1.SmsService", func(s *grpc.Server) {
		smsv1.RegisterSmsServiceServer(s, NewSmsGRPCServer(smsSvc))
	})

	return smsSvc
}
