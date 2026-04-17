package service

import (
	"context"
	"testing"

	"github.com/parkhub/api/internal/domains/iot/domain"
	"github.com/parkhub/api/internal/domains/iot/errs"
	repomocks "github.com/parkhub/api/internal/domains/iot/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	go_mock "go.uber.org/mock/gomock"
)

func newTestSvc(ctrl *go_mock.Controller) (*deviceService, *repomocks.MockDeviceRepo) {
	repo := repomocks.NewMockDeviceRepo(ctrl)
	return NewDeviceService(repo).(*deviceService), repo
}

func TestDeviceService_Create(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().Create(go_mock.Any(), go_mock.Any()).Return(nil)

	d, err := svc.Create(context.Background(), &CreateDeviceRequest{
		TenantID:        "t1",
		ID:              "DEV-001",
		Name:            "测试设备",
		Type:            domain.DeviceTypeIntegrated,
		FirmwareVersion: "v1.0",
	})
	assert.NoError(t, err)
	assert.Equal(t, "DEV-001", d.ID)
	assert.Equal(t, domain.DeviceStatusPending, d.Status)
}

func TestDeviceService_GetByID(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001"}, nil)

	d, err := svc.GetByID(context.Background(), &GetDeviceRequest{TenantID: "t1", ID: "DEV-001"})
	assert.NoError(t, err)
	assert.Equal(t, "DEV-001", d.ID)
}

func TestDeviceService_List(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().List(go_mock.Any(), go_mock.Any(), 1, 20).Return([]*domain.Device{{ID: "DEV-001"}}, int64(1), nil)

	resp, err := svc.List(context.Background(), &ListDevicesRequest{TenantID: "t1"})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), resp.Total)
	assert.Len(t, resp.Devices, 1)
}

func TestDeviceService_UpdateName(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001", Name: "旧"}, nil)
	repo.EXPECT().Update(go_mock.Any(), go_mock.Any()).Return(nil)

	d, err := svc.UpdateName(context.Background(), &UpdateDeviceNameRequest{TenantID: "t1", ID: "DEV-001", Name: "新"})
	assert.NoError(t, err)
	assert.Equal(t, "新", d.Name)
}

func TestDeviceService_Bind(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001", Status: domain.DeviceStatusPending}, nil)
	repo.EXPECT().Update(go_mock.Any(), go_mock.Any()).Return(nil)

	d, err := svc.Bind(context.Background(), &BindDeviceRequest{TenantID: "t1", ID: "DEV-001", ParkingLotID: "lot-1", GateID: "gate-1"})
	assert.NoError(t, err)
	assert.Equal(t, "lot-1", *d.ParkingLotID)
	assert.Equal(t, domain.DeviceStatusActive, d.Status)
}

func TestDeviceService_Unbind(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	lotID := "lot-1"
	gateID := "gate-1"
	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001", ParkingLotID: &lotID, GateID: &gateID}, nil)
	repo.EXPECT().Update(go_mock.Any(), go_mock.Any()).Return(nil)

	d, err := svc.Unbind(context.Background(), &UnbindDeviceRequest{TenantID: "t1", ID: "DEV-001"})
	assert.NoError(t, err)
	assert.Nil(t, d.ParkingLotID)
}

func TestDeviceService_Disable(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001", Status: domain.DeviceStatusActive}, nil)
	repo.EXPECT().Update(go_mock.Any(), go_mock.Any()).Return(nil)

	d, err := svc.Disable(context.Background(), &ChangeDeviceStatusRequest{TenantID: "t1", ID: "DEV-001"})
	assert.NoError(t, err)
	assert.Equal(t, domain.DeviceStatusDisabled, d.Status)
}

func TestDeviceService_Enable(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001", Status: domain.DeviceStatusDisabled}, nil)
	repo.EXPECT().Update(go_mock.Any(), go_mock.Any()).Return(nil)

	d, err := svc.Enable(context.Background(), &ChangeDeviceStatusRequest{TenantID: "t1", ID: "DEV-001"})
	assert.NoError(t, err)
	assert.Equal(t, domain.DeviceStatusActive, d.Status)
}

func TestDeviceService_Delete(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001"}, nil)
	repo.EXPECT().Delete(go_mock.Any(), "t1", "DEV-001").Return(nil)

	err := svc.Delete(context.Background(), &DeleteDeviceRequest{TenantID: "t1", ID: "DEV-001"})
	assert.NoError(t, err)
}

func TestDeviceService_Delete_Bound(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	lotID := "lot-1"
	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001", ParkingLotID: &lotID}, nil)

	err := svc.Delete(context.Background(), &DeleteDeviceRequest{TenantID: "t1", ID: "DEV-001"})
	assert.ErrorIs(t, err, errs.ErrDeviceMustUnbind)
}

func TestDeviceService_GetStats(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().CountByStatus(go_mock.Any(), "t1").Return(int64(2), int64(5), int64(1), int64(3), nil)

	stats, err := svc.GetStats(context.Background(), "t1")
	assert.NoError(t, err)
	assert.Equal(t, int64(11), stats.Total)
	assert.Equal(t, int64(5), stats.Active)
	assert.Equal(t, int64(1), stats.Offline)
	assert.Equal(t, int64(2), stats.Pending)
	assert.Equal(t, int64(3), stats.Disabled)
}

func TestDeviceService_SendCommand_Up(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001", Status: domain.DeviceStatusActive}, nil)

	resp, err := svc.SendCommand(context.Background(), "t1", "DEV-001", "up")
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Message, "抬杆")
}

func TestDeviceService_SendCommand_Offline(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().GetByID(go_mock.Any(), "t1", "DEV-001").Return(&domain.Device{ID: "DEV-001", Status: domain.DeviceStatusOffline}, nil)

	_, err := svc.SendCommand(context.Background(), "t1", "DEV-001", "up")
	assert.ErrorIs(t, err, errs.ErrDeviceOffline)
}

func TestDeviceService_BatchDisable(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().UpdateStatusBatch(go_mock.Any(), "t1", []string{"DEV-001", "DEV-002"}, "disabled").Return(int64(2), nil)

	err := svc.BatchDisable(context.Background(), &BatchChangeDeviceStatusRequest{TenantID: "t1", IDs: []string{"DEV-001", "DEV-002"}})
	assert.NoError(t, err)
}

func TestDeviceService_BatchDelete(t *testing.T) {
	ctrl := go_mock.NewController(t)
	defer ctrl.Finish()
	svc, repo := newTestSvc(ctrl)

	repo.EXPECT().UnbindByDeviceIDs(go_mock.Any(), "t1", []string{"DEV-001"}).Return(nil)
	repo.EXPECT().DeleteBatch(go_mock.Any(), "t1", []string{"DEV-001"}).Return(int64(1), nil)

	err := svc.BatchDelete(context.Background(), &BatchDeleteDeviceRequest{TenantID: "t1", IDs: []string{"DEV-001"}})
	require.NoError(t, err)
}
