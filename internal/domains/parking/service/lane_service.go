package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	iotdomain "github.com/parkhub/api/internal/domains/iot/domain"
	iotservice "github.com/parkhub/api/internal/domains/iot/service"
	"github.com/parkhub/api/internal/domains/parking/domain"
	"github.com/parkhub/api/internal/domains/parking/repository"
)

type laneService struct {
	laneRepo  repository.LaneRepo
	deviceSvc iotservice.DeviceService
}

func NewLaneService(laneRepo repository.LaneRepo, deviceSvc iotservice.DeviceService) LaneService {
	return &laneService{laneRepo: laneRepo, deviceSvc: deviceSvc}
}

func (s *laneService) GetLaneConfig(ctx context.Context, tenantID, parkingLotID string) (*LaneConfigResponse, error) {
	lanes, err := s.laneRepo.ListByParkingLotID(ctx, tenantID, parkingLotID)
	if err != nil {
		return nil, err
	}

	// Get available (unbound) devices for this tenant
	devicesResp, err := s.deviceSvc.List(ctx, &iotservice.ListDevicesRequest{
		TenantID: tenantID,
		Page:     1,
		PageSize: 200,
	})
	if err != nil {
		slog.Warn("failed to list devices for lane config", "error", err)
	}

	// Build map of device IDs already bound to lanes
	boundDevices := make(map[string]bool)
	for _, l := range lanes {
		if l.DeviceID != nil {
			boundDevices[*l.DeviceID] = true
		}
	}

	// Available devices = unbound + not disabled
	var available []*AvailableDevice
	if devicesResp != nil {
		for _, d := range devicesResp.Devices {
			if boundDevices[d.ID] || d.Status == iotdomain.DeviceStatusDisabled {
				continue
			}
			available = append(available, &AvailableDevice{
				ID:     d.ID,
				Name:   d.Name,
				Status: string(d.Status),
			})
		}
	}

	// Build LaneWithDevice by looking up device info
	lanesWithDevices := make([]*domain.LaneWithDevice, 0, len(lanes))
	deviceMap := make(map[string]*iotdomain.Device)
	if devicesResp != nil {
		for _, d := range devicesResp.Devices {
			deviceMap[d.ID] = d
		}
	}

	for _, l := range lanes {
		ld := &domain.LaneWithDevice{Lane: *l}
		if l.DeviceID != nil {
			if dev, ok := deviceMap[*l.DeviceID]; ok {
				ld.Device = &domain.LaneDeviceInfo{
					ID:            dev.ID,
					Name:          dev.Name,
					Status:        string(dev.Status),
					LastHeartbeat: dev.LastHeartbeat,
				}
			}
		}
		lanesWithDevices = append(lanesWithDevices, ld)
	}

	return &LaneConfigResponse{
		Lanes:            lanesWithDevices,
		AvailableDevices: available,
	}, nil
}

func (s *laneService) UpdateLanes(ctx context.Context, tenantID string, req *UpdateLanesRequest) (*UpdateLanesResponse, error) {
	// Load current state
	currentLanes, err := s.laneRepo.ListByParkingLotID(ctx, tenantID, req.ParkingLotID)
	if err != nil {
		return nil, err
	}

	currentMap := make(map[string]*domain.Lane)
	for _, l := range currentLanes {
		currentMap[l.ID] = l
	}

	// Build sets
	requestIDs := make(map[string]bool)
	for _, input := range req.Lanes {
		if input.ID != nil && *input.ID != "" {
			requestIDs[*input.ID] = true
		}
	}

	// Phase 1: Delete lanes not in request (unbind devices first)
	for id, lane := range currentMap {
		if !requestIDs[id] {
			if lane.DeviceID != nil {
				if _, err := s.deviceSvc.Unbind(ctx, &iotservice.UnbindDeviceRequest{
					TenantID: tenantID,
					ID:       *lane.DeviceID,
				}); err != nil {
					slog.Warn("failed to unbind device during lane deletion", "deviceID", *lane.DeviceID, "error", err)
				}
			}
			if err := s.laneRepo.Delete(ctx, tenantID, id); err != nil {
				return nil, fmt.Errorf("delete lane %s: %w", id, err)
			}
		}
	}

	// Phase 2: Create and Update
	for _, input := range req.Lanes {
		if input.ID == nil || *input.ID == "" {
			// Create new lane
			exists, err := s.laneRepo.ExistsByName(ctx, req.ParkingLotID, input.Name)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, fmt.Errorf("lane name '%s' already exists", input.Name)
			}
			lane := domain.NewLane(uuid.New().String(), tenantID, req.ParkingLotID, input.Name, input.Type)
			if err := s.laneRepo.Create(ctx, lane); err != nil {
				return nil, fmt.Errorf("create lane: %w", err)
			}
			if input.DeviceID != nil && *input.DeviceID != "" {
				if err := s.bindDevice(ctx, tenantID, *input.DeviceID, req.ParkingLotID, lane.ID); err != nil {
					slog.Warn("failed to bind device to new lane", "deviceID", *input.DeviceID, "error", err)
				} else {
					lane.SetDevice(*input.DeviceID)
					_ = s.laneRepo.Update(ctx, lane)
				}
			}
		} else {
			// Update existing lane
			lane, ok := currentMap[*input.ID]
			if !ok {
				return nil, fmt.Errorf("lane %s not found", *input.ID)
			}

			oldDeviceID := lane.DeviceID
			newDeviceID := input.DeviceID

			// Handle device binding changes
			if !deviceIDEqual(oldDeviceID, newDeviceID) {
				// Unbind old device if exists
				if oldDeviceID != nil {
					_, _ = s.deviceSvc.Unbind(ctx, &iotservice.UnbindDeviceRequest{
						TenantID: tenantID,
						ID:       *oldDeviceID,
					})
				}
				// Bind new device if specified
				if newDeviceID != nil && *newDeviceID != "" {
					if err := s.bindDevice(ctx, tenantID, *newDeviceID, req.ParkingLotID, lane.ID); err != nil {
						slog.Warn("failed to bind device to lane", "deviceID", *newDeviceID, "error", err)
						lane.ClearDevice()
					} else {
						lane.SetDevice(*newDeviceID)
					}
				} else {
					lane.ClearDevice()
				}
			}

			lane.Name = input.Name
			if err := s.laneRepo.Update(ctx, lane); err != nil {
				return nil, fmt.Errorf("update lane %s: %w", *input.ID, err)
			}
		}
	}

	// Reload final state
	finalLanes, err := s.laneRepo.ListByParkingLotID(ctx, tenantID, req.ParkingLotID)
	if err != nil {
		return nil, err
	}

	lanesWithDevices := make([]*domain.LaneWithDevice, 0, len(finalLanes))
	for _, l := range finalLanes {
		lanesWithDevices = append(lanesWithDevices, &domain.LaneWithDevice{Lane: *l})
	}

	return &UpdateLanesResponse{Lanes: lanesWithDevices}, nil
}

func (s *laneService) bindDevice(ctx context.Context, tenantID, deviceID, parkingLotID, laneID string) error {
	_, err := s.deviceSvc.Bind(ctx, &iotservice.BindDeviceRequest{
		TenantID:     tenantID,
		ID:           deviceID,
		ParkingLotID: parkingLotID,
		GateID:       laneID,
	})
	return err
}

func deviceIDEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
