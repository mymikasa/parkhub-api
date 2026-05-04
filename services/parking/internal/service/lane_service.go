package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/parkhub/api/services/parking/internal/domain"
	"github.com/parkhub/api/services/parking/internal/repository"
)

const deviceStatusDisabled = "disabled"

type laneService struct {
	laneRepo  repository.LaneRepo
	deviceSvc *IoTDeviceClient
}

func NewLaneService(laneRepo repository.LaneRepo, deviceSvc *IoTDeviceClient) LaneService {
	return &laneService{laneRepo: laneRepo, deviceSvc: deviceSvc}
}

func (s *laneService) GetLaneConfig(ctx context.Context, tenantID, parkingLotID string) (*LaneConfigResponse, error) {
	lanes, err := s.laneRepo.ListByParkingLotID(ctx, tenantID, parkingLotID)
	if err != nil {
		return nil, err
	}

	devices, err := s.deviceSvc.ListDevices(ctx, tenantID, 1, 200)
	if err != nil {
		slog.Warn("failed to list devices for lane config", "error", err)
	}

	boundDevices := make(map[string]bool)
	for _, l := range lanes {
		if l.DeviceID != nil {
			boundDevices[*l.DeviceID] = true
		}
	}

	available := make([]*AvailableDevice, 0)
	if devices != nil {
		for _, d := range devices {
			if boundDevices[d.ID] || d.Status == deviceStatusDisabled {
				continue
			}
			available = append(available, &AvailableDevice{
				ID:     d.ID,
				Name:   d.Name,
				Status: d.Status,
			})
		}
	}

	lanesWithDevices := make([]*domain.LaneWithDevice, 0, len(lanes))
	deviceMap := make(map[string]*DeviceInfo)
	if devices != nil {
		for _, d := range devices {
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
					Status:        dev.Status,
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
	currentLanes, err := s.laneRepo.ListByParkingLotID(ctx, tenantID, req.ParkingLotID)
	if err != nil {
		return nil, err
	}

	currentMap := make(map[string]*domain.Lane)
	for _, l := range currentLanes {
		currentMap[l.ID] = l
	}

	requestIDs := make(map[string]bool)
	for _, input := range req.Lanes {
		if input.ID != nil && *input.ID != "" {
			requestIDs[*input.ID] = true
		}
	}

	for id, lane := range currentMap {
		if !requestIDs[id] {
			if lane.DeviceID != nil {
				if err := s.deviceSvc.UnbindDevice(ctx, tenantID, *lane.DeviceID); err != nil {
					slog.Warn("failed to unbind device during lane deletion", "deviceID", *lane.DeviceID, "error", err)
				}
			}
			if err := s.laneRepo.Delete(ctx, tenantID, id); err != nil {
				return nil, fmt.Errorf("delete lane %s: %w", id, err)
			}
		}
	}

	for _, input := range req.Lanes {
		if input.ID == nil || *input.ID == "" {
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
				if err := s.deviceSvc.BindDevice(ctx, tenantID, *input.DeviceID, req.ParkingLotID, lane.ID); err != nil {
					slog.Warn("failed to bind device to new lane", "deviceID", *input.DeviceID, "error", err)
				} else {
					lane.SetDevice(*input.DeviceID)
					_ = s.laneRepo.Update(ctx, lane)
				}
			}
		} else {
			lane, ok := currentMap[*input.ID]
			if !ok {
				return nil, fmt.Errorf("lane %s not found", *input.ID)
			}

			oldDeviceID := lane.DeviceID
			newDeviceID := input.DeviceID

			if !deviceIDEqual(oldDeviceID, newDeviceID) {
				if oldDeviceID != nil {
					_ = s.deviceSvc.UnbindDevice(ctx, tenantID, *oldDeviceID)
				}
				if newDeviceID != nil && *newDeviceID != "" {
					if err := s.deviceSvc.BindDevice(ctx, tenantID, *newDeviceID, req.ParkingLotID, lane.ID); err != nil {
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

func deviceIDEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
