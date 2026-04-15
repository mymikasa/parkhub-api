package grpc

import (
	"github.com/parkhub/api/internal/domains/parking/repository"
	"github.com/parkhub/api/internal/domains/parking/repository/dao"
	"github.com/parkhub/api/internal/domains/parking/service"
	parkingv1 "github.com/parkhub/api/internal/gen/api/proto/parking/v1"
	"github.com/parkhub/api/internal/registry"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

func RegisterServices(reg *registry.Registry, db *gorm.DB) {
	lotDAO := dao.NewParkingLotDAO(db)
	lotRepo := repository.NewParkingLotRepo(lotDAO)
	lotSvc := service.NewParkingLotService(lotRepo)

	reg.MustRegister("parking.v1.ParkingLotService", func(s *grpc.Server) {
		parkingv1.RegisterParkingLotServiceServer(s, NewParkingLotGRPCServer(lotSvc))
	})
}
