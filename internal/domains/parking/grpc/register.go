package grpc

import (
	"github.com/parkhub/api/internal/domains/parking/repository"
	"github.com/parkhub/api/internal/domains/parking/repository/dao"
	"github.com/parkhub/api/internal/domains/parking/service"
	iotservice "github.com/parkhub/api/internal/domains/iot/service"
	parkingv1 "github.com/parkhub/api/internal/gen/api/proto/parking/v1"
	"github.com/parkhub/api/internal/registry"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func RegisterServices(reg *registry.Registry, db *gorm.DB, deviceSvc iotservice.DeviceService) {
	lotDAO := dao.NewParkingLotDAO(db)
	lotRepo := repository.NewParkingLotRepo(lotDAO)

	laneDAO := dao.NewLaneDAO(db)
	laneRepo := repository.NewLaneRepo(laneDAO)

	lotSvc := service.NewParkingLotService(lotRepo, laneRepo)

	reg.MustRegister("parking.v1.ParkingLotService", func(s *grpc.Server) {
		parkingv1.RegisterParkingLotServiceServer(s, NewParkingLotGRPCServer(lotSvc))
	})

	if deviceSvc != nil {
		laneSvc := service.NewLaneService(laneRepo, deviceSvc)

		reg.MustRegister("parking.v1.LaneService", func(s *grpc.Server) {
			parkingv1.RegisterLaneServiceServer(s, NewLaneGRPCServer(laneSvc))
		})
	}
}
