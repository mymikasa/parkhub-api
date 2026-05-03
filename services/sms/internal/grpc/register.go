package grpc

import (
	"github.com/parkhub/api/services/sms/internal/gateway"
	smsv1 "github.com/parkhub/api/services/sms/internal/gen/api/proto/sms/v1"
	"github.com/parkhub/api/services/sms/internal/registry"
	"github.com/parkhub/api/services/sms/internal/repository"
	"github.com/parkhub/api/services/sms/internal/repository/cache"
	"github.com/parkhub/api/services/sms/internal/repository/dao"
	"github.com/parkhub/api/services/sms/internal/service"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func RegisterServices(reg *registry.Registry, db *gorm.DB, rdb *redis.Client) {
	smsDAO := dao.NewSmsRecordDAO(db)
	smsCache := cache.NewRedisSmsCache(rdb)
	smsRepo := repository.NewSmsRepository(smsDAO, smsCache)
	smsGateway := gateway.NewMockSmsGateway()
	smsSvc := service.NewSmsService(smsRepo, smsGateway)

	reg.MustRegister("sms.v1.SmsService", func(s *grpc.Server) {
		smsv1.RegisterSmsServiceServer(s, NewSmsGRPCServer(smsSvc))
	})

	_ = smsSvc
}
