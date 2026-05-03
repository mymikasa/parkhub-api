package client

import (
	"context"

	parkingv1 "github.com/parkhub/api/services/identity/internal/gen/api/proto/parking/v1"
	"github.com/parkhub/api/services/identity/internal/service"
)

var _ service.ParkingLotCounter = (*ParkingLotCounterClient)(nil)

type ParkingLotCounterClient struct {
	client parkingv1.ParkingLotServiceClient
}

func NewParkingLotCounterClient(client parkingv1.ParkingLotServiceClient) service.ParkingLotCounter {
	return &ParkingLotCounterClient{client: client}
}

func (c *ParkingLotCounterClient) CountParkingLots(ctx context.Context) (int64, error) {
	resp, err := c.client.ListParkingLots(ctx, &parkingv1.ListParkingLotsRequest{
		Pagination: &parkingv1.PaginationRequest{
			Page:     1,
			PageSize: 1,
		},
	})
	if err != nil {
		return 0, err
	}
	if resp.Pagination != nil {
		return resp.Pagination.Total, nil
	}
	return int64(len(resp.ParkingLots)), nil
}
