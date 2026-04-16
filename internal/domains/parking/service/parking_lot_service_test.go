package service

import (
	"context"
	"testing"

	"github.com/parkhub/api/internal/domains/parking/domain"
	"github.com/parkhub/api/internal/domains/parking/errs"
	"github.com/parkhub/api/internal/domains/parking/repository"
	repomocks "github.com/parkhub/api/internal/domains/parking/repository/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newTestParkingLot(tenantID, id string) *domain.ParkingLot {
	return &domain.ParkingLot{
		ID:              id,
		TenantID:        tenantID,
		Name:            "Test Lot",
		Address:         "123 Test St",
		TotalSpaces:     200,
		AvailableSpaces: 180,
		LotType:         domain.LotTypeGround,
		Status:          domain.ParkingLotStatusActive,
		CreatedAt:       1000,
		UpdatedAt:       1000,
	}
}

func TestParkingLotService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	mockRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

	lot, err := svc.Create(ctx, &CreateParkingLotRequest{
		TenantID:    "tenant-1",
		Name:        "朝阳停车场",
		Address:     "朝阳区xxx",
		TotalSpaces: 200,
		LotType:     domain.LotTypeUnderground,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, lot.ID)
	assert.Equal(t, "tenant-1", lot.TenantID)
	assert.Equal(t, "朝阳停车场", lot.Name)
	assert.Equal(t, 200, lot.TotalSpaces)
	assert.Equal(t, 200, lot.AvailableSpaces)
	assert.Equal(t, domain.ParkingLotStatusActive, lot.Status)
	assert.Equal(t, domain.LotTypeUnderground, lot.LotType)
}

func TestParkingLotService_Create_DuplicateName(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	mockRepo.EXPECT().Create(ctx, gomock.Any()).Return(errs.ErrParkingLotNameDuplicate)

	_, err := svc.Create(ctx, &CreateParkingLotRequest{
		TenantID: "tenant-1", Name: "dup", Address: "addr", TotalSpaces: 100, LotType: domain.LotTypeGround,
	})
	assert.ErrorIs(t, err, errs.ErrParkingLotNameDuplicate)
}

func TestParkingLotService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	expected := newTestParkingLot("tenant-1", "lot-1")
	mockRepo.EXPECT().GetByID(ctx, "tenant-1", "lot-1").Return(expected, nil)

	lot, err := svc.GetByID(ctx, "tenant-1", "lot-1")
	assert.NoError(t, err)
	assert.Equal(t, "lot-1", lot.ID)
}

func TestParkingLotService_GetByID_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(ctx, "tenant-1", "nonexistent").Return(nil, errs.ErrParkingLotNotFound)

	_, err := svc.GetByID(ctx, "tenant-1", "nonexistent")
	assert.ErrorIs(t, err, errs.ErrParkingLotNotFound)
}

func TestParkingLotService_List_DefaultPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	mockRepo.EXPECT().List(ctx, gomock.Any(), 1, 20).Return([]*domain.ParkingLot{}, int64(0), nil)

	resp, err := svc.List(ctx, &ListParkingLotsRequest{TenantID: "tenant-1"})
	assert.NoError(t, err)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
}

func TestParkingLotService_List_PageSizeCap(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	mockRepo.EXPECT().List(ctx, gomock.Any(), 1, 100).Return([]*domain.ParkingLot{}, int64(0), nil)

	resp, err := svc.List(ctx, &ListParkingLotsRequest{TenantID: "tenant-1", PageSize: 200})
	assert.NoError(t, err)
	assert.Equal(t, 100, resp.PageSize)
}

func TestParkingLotService_List_TotalPages(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	lots := make([]*domain.ParkingLot, 2)
	mockRepo.EXPECT().List(ctx, gomock.Any(), 1, 2).Return(lots, int64(5), nil)

	resp, err := svc.List(ctx, &ListParkingLotsRequest{TenantID: "tenant-1", PageSize: 2})
	assert.NoError(t, err)
	assert.Equal(t, 3, resp.TotalPages)
}

func TestParkingLotService_List_FilterConversion(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	mockRepo.EXPECT().List(ctx, repository.ParkingLotFilter{
		TenantID: "tenant-1",
		Status:   domain.ParkingLotStatusActive,
		LotType:  domain.LotTypeGround,
		Keyword:  "朝阳",
	}, 1, 20).Return([]*domain.ParkingLot{}, int64(0), nil)

	_, err := svc.List(ctx, &ListParkingLotsRequest{
		TenantID: "tenant-1",
		Status:   domain.ParkingLotStatusActive,
		LotType:  domain.LotTypeGround,
		Keyword:  "朝阳",
	})
	assert.NoError(t, err)
}

func TestParkingLotService_Update_PartialFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	existing := newTestParkingLot("tenant-1", "lot-1")
	mockRepo.EXPECT().GetByID(ctx, "tenant-1", "lot-1").Return(existing, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

	newName := "新名字"
	lot, err := svc.Update(ctx, &UpdateParkingLotRequest{
		ID:       "lot-1",
		TenantID: "tenant-1",
		Name:     &newName,
	})
	assert.NoError(t, err)
	assert.Equal(t, "新名字", lot.Name)
	assert.Equal(t, "123 Test St", lot.Address)
}

func TestParkingLotService_Update_StatusTransition(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	lot := newTestParkingLot("tenant-1", "lot-1")
	mockRepo.EXPECT().GetByID(ctx, "tenant-1", "lot-1").Return(lot, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

	inactive := domain.ParkingLotStatusInactive
	result, err := svc.Update(ctx, &UpdateParkingLotRequest{
		ID:       "lot-1",
		TenantID: "tenant-1",
		Status:   &inactive,
	})
	assert.NoError(t, err)
	assert.Equal(t, domain.ParkingLotStatusInactive, result.Status)
}

func TestParkingLotService_Update_StatusInvalidTransition(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	lot := newTestParkingLot("tenant-1", "lot-1")
	lot.Status = domain.ParkingLotStatusInactive
	mockRepo.EXPECT().GetByID(ctx, "tenant-1", "lot-1").Return(lot, nil)

	inactive := domain.ParkingLotStatusInactive
	_, err := svc.Update(ctx, &UpdateParkingLotRequest{
		ID:       "lot-1",
		TenantID: "tenant-1",
		Status:   &inactive,
	})
	assert.ErrorIs(t, err, errs.ErrParkingLotInvalidStatus)
}

func TestParkingLotService_Update_TotalSpacesAdjustsAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	lot := newTestParkingLot("tenant-1", "lot-1")
	lot.TotalSpaces = 200
	lot.AvailableSpaces = 180
	mockRepo.EXPECT().GetByID(ctx, "tenant-1", "lot-1").Return(lot, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

	newTotal := 250
	result, err := svc.Update(ctx, &UpdateParkingLotRequest{
		ID:          "lot-1",
		TenantID:    "tenant-1",
		TotalSpaces: &newTotal,
	})
	assert.NoError(t, err)
	assert.Equal(t, 250, result.TotalSpaces)
	assert.Equal(t, 230, result.AvailableSpaces)
}

func TestParkingLotService_Update_TotalSpacesTooSmall(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	lot := newTestParkingLot("tenant-1", "lot-1")
	lot.TotalSpaces = 200
	lot.AvailableSpaces = 180
	mockRepo.EXPECT().GetByID(ctx, "tenant-1", "lot-1").Return(lot, nil)

	newTotal := 10
	_, err := svc.Update(ctx, &UpdateParkingLotRequest{
		ID:          "lot-1",
		TenantID:    "tenant-1",
		TotalSpaces: &newTotal,
	})
	assert.ErrorIs(t, err, errs.ErrParkingLotInvalidCapacity)
}

func TestParkingLotService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	mockRepo.EXPECT().Delete(ctx, "tenant-1", "lot-1").Return(nil)

	err := svc.Delete(ctx, "tenant-1", "lot-1")
	assert.NoError(t, err)
}

func TestParkingLotService_GetStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	mockRepo.EXPECT().SumStats(ctx, "tenant-1").Return(int64(300), int64(230), nil)

	stats, err := svc.GetStats(ctx, "tenant-1")
	assert.NoError(t, err)
	assert.Equal(t, int64(300), stats.TotalSpaces)
	assert.Equal(t, int64(230), stats.AvailableSpaces)
	assert.Equal(t, int64(70), stats.OccupiedVehicles)
	assert.Equal(t, int64(0), stats.TotalGates)
}

func TestParkingLotService_GetStats_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := repomocks.NewMockParkingLotRepo(ctrl)
	svc := NewParkingLotService(mockRepo)
	ctx := context.Background()

	mockRepo.EXPECT().SumStats(ctx, "tenant-1").Return(int64(0), int64(0), nil)

	stats, err := svc.GetStats(ctx, "tenant-1")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), stats.OccupiedVehicles)
}
