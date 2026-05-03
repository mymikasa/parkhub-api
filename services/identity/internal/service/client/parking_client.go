package client

import (
	"context"

	"github.com/parkhub/api/services/identity/internal/service"
)

var _ service.ParkingLotCounter = (*NoopParkingLotCounter)(nil)

type NoopParkingLotCounter struct{}

func NewNoopParkingLotCounter() service.ParkingLotCounter {
	return &NoopParkingLotCounter{}
}

func (n *NoopParkingLotCounter) CountParkingLots(_ context.Context) (int64, error) {
	return 0, nil
}
