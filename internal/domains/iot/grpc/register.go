package grpc

import (
	"github.com/parkhub/api/internal/domains/iot/repository"
	"github.com/parkhub/api/internal/domains/iot/repository/dao"
	"github.com/parkhub/api/internal/domains/iot/service"
	iotv1 "github.com/parkhub/api/internal/gen/api/proto/iot/v1"
	"github.com/parkhub/api/internal/registry"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func RegisterServices(reg *registry.Registry, db *gorm.DB) {
	deviceDAO := dao.NewDeviceDAO(db)
	deviceRepo := repository.NewDeviceRepo(deviceDAO)
	deviceSvc := service.NewDeviceService(deviceRepo)

	reg.MustRegister("iot.v1.DeviceService", func(s *grpc.Server) {
		iotv1.RegisterDeviceServiceServer(s, NewDeviceGRPCServer(deviceSvc))
	})
}
