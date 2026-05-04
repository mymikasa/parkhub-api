package grpc

import (
	parkingv1 "github.com/parkhub/api/services/parking/internal/gen/api/proto/parking/v1"
	"github.com/parkhub/api/services/parking/internal/registry"
	"github.com/parkhub/api/services/parking/internal/repository"
	"github.com/parkhub/api/services/parking/internal/repository/dao"
	"github.com/parkhub/api/services/parking/internal/service"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func RegisterServices(reg *registry.Registry, db *gorm.DB, iotClient *service.IoTDeviceClient) {
	lotDAO := dao.NewParkingLotDAO(db)
	lotRepo := repository.NewParkingLotRepo(lotDAO)

	laneDAO := dao.NewLaneDAO(db)
	laneRepo := repository.NewLaneRepo(laneDAO)

	lotSvc := service.NewParkingLotService(lotRepo, laneRepo)

	reg.MustRegister("parking.v1.ParkingLotService", func(s *grpc.Server) {
		parkingv1.RegisterParkingLotServiceServer(s, NewParkingLotGRPCServer(lotSvc))
	})

	laneSvc := service.NewLaneService(laneRepo, iotClient)

	reg.MustRegister("parking.v1.LaneService", func(s *grpc.Server) {
		parkingv1.RegisterLaneServiceServer(s, NewLaneGRPCServer(laneSvc))
	})
}
