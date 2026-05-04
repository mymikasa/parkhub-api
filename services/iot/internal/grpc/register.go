package grpc

import (
	iotv1 "github.com/parkhub/api/services/iot/internal/gen/api/proto/iot/v1"
	"github.com/parkhub/api/services/iot/internal/registry"
	"github.com/parkhub/api/services/iot/internal/repository"
	"github.com/parkhub/api/services/iot/internal/repository/dao"
	"github.com/parkhub/api/services/iot/internal/service"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func RegisterServices(reg *registry.Registry, db *gorm.DB) {
	deviceDAO := dao.NewDeviceDAO(db)
	deviceRepo := repository.NewDeviceRepo(deviceDAO, db)
	deviceSvc := service.NewDeviceService(deviceRepo)

	reg.MustRegister("iot.v1.DeviceService", func(s *grpc.Server) {
		iotv1.RegisterDeviceServiceServer(s, NewDeviceGRPCServer(deviceSvc))
	})
}
