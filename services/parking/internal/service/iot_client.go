package service

import (
	"context"
	"time"

	commonv1 "github.com/parkhub/api/services/parking/internal/gen/api/proto/common/v1"
	iotv1 "github.com/parkhub/api/services/parking/internal/gen/api/proto/iot/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DeviceInfo struct {
	ID            string
	Name          string
	Status        string
	LastHeartbeat *time.Time
	ParkingLotID  *string
	GateID        *string
}

type IoTDeviceClient struct {
	client iotv1.DeviceServiceClient
	conn   *grpc.ClientConn
}

func NewIoTDeviceClient(addr string) (*IoTDeviceClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return &IoTDeviceClient{
		client: iotv1.NewDeviceServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *IoTDeviceClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *IoTDeviceClient) ListDevices(ctx context.Context, tenantID string, page, pageSize int32) ([]*DeviceInfo, error) {
	resp, err := c.client.ListDevices(ctx, &iotv1.ListDevicesRequest{
		Pagination: &commonv1.PaginationRequest{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	devices := make([]*DeviceInfo, 0, len(resp.Devices))
	for _, d := range resp.Devices {
		info := &DeviceInfo{
			ID:     d.Id,
			Name:   d.Name,
			Status: d.Status.String(),
		}
		if d.LastHeartbeat != nil {
			t := d.LastHeartbeat.AsTime()
			info.LastHeartbeat = &t
		}
		if d.ParkingLotId != nil {
			info.ParkingLotID = d.ParkingLotId
		}
		if d.GateId != nil {
			info.GateID = d.GateId
		}
		devices = append(devices, info)
	}
	return devices, nil
}

func (c *IoTDeviceClient) BindDevice(ctx context.Context, tenantID, deviceID, parkingLotID, gateID string) error {
	_, err := c.client.BindDevice(ctx, &iotv1.BindDeviceRequest{
		Id:           deviceID,
		ParkingLotId: parkingLotID,
		GateId:       gateID,
	})
	return err
}

func (c *IoTDeviceClient) UnbindDevice(ctx context.Context, tenantID, deviceID string) error {
	_, err := c.client.UnbindDevice(ctx, &iotv1.UnbindDeviceRequest{
		Id: deviceID,
	})
	return err
}
