package dao

import (
	"context"
	"strings"

	"github.com/parkhub/api/internal/domains/iot/errs"

	"gorm.io/gorm"
)

type Device struct {
	ID              string  `gorm:"primaryKey;type:varchar(64)"`
	TenantID        string  `gorm:"type:varchar(36);index"`
	Name            string  `gorm:"type:varchar(100)"`
	Type            string  `gorm:"type:varchar(20)"`
	Status          string  `gorm:"type:varchar(20);index"`
	FirmwareVersion string  `gorm:"type:varchar(50)"`
	LastHeartbeatAt *int64  `gorm:"type:bigint"`
	ParkingLotID    *string `gorm:"type:varchar(36);index"`
	GateID          *string `gorm:"type:varchar(36)"`
	CreatedAt       int64   `gorm:"autoCreateTime:milli"`
	UpdatedAt       int64   `gorm:"autoUpdateTime:milli"`
}

func (Device) TableName() string {
	return "devices"
}

type DeviceFilter struct {
	TenantID     string
	Status       string
	ParkingLotID string
	Keyword      string
}

//go:generate mockgen -source=./device.go -package=daomocks -destination=./mocks/device.mock.go DeviceDAO

type DeviceDAO interface {
	Insert(ctx context.Context, device *Device) error
	FindByID(ctx context.Context, id string) (*Device, error)
	FindAll(ctx context.Context, filter DeviceFilter, page, pageSize int) ([]*Device, int64, error)
	Update(ctx context.Context, device *Device) error
	Delete(ctx context.Context, id string) error
	DeleteBatch(ctx context.Context, ids []string) (int64, error)
	CountByStatus(ctx context.Context, tenantID string) (pending, active, offline, disabled int64, err error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateStatusBatch(ctx context.Context, ids []string, status string) (int64, error)
	UnbindByDeviceIDs(ctx context.Context, ids []string) error
}

type GORMDeviceDAO struct {
	db *gorm.DB
}

func NewDeviceDAO(db *gorm.DB) DeviceDAO {
	return &GORMDeviceDAO{db: db}
}

func (d *GORMDeviceDAO) Insert(ctx context.Context, device *Device) error {
	err := d.db.WithContext(ctx).Create(device).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "Duplicate") {
			return errs.ErrDeviceIDDuplicate
		}
		return err
	}
	return nil
}

func (d *GORMDeviceDAO) FindByID(ctx context.Context, id string) (*Device, error) {
	var device Device
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&device).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrDeviceNotFound
		}
		return nil, err
	}
	return &device, nil
}

func (d *GORMDeviceDAO) FindAll(ctx context.Context, filter DeviceFilter, page, pageSize int) ([]*Device, int64, error) {
	query := d.db.WithContext(ctx).Model(&Device{})

	if filter.TenantID != "" {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.ParkingLotID != "" {
		query = query.Where("parking_lot_id = ?", filter.ParkingLotID)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR id LIKE ?", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var devices []*Device
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&devices).Error; err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

func (d *GORMDeviceDAO) Update(ctx context.Context, device *Device) error {
	result := d.db.WithContext(ctx).Model(&Device{}).
		Where("id = ?", device.ID).
		Select("name", "type", "status", "firmware_version", "last_heartbeat_at", "parking_lot_id", "gate_id", "updated_at").
		Updates(device)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrDeviceNotFound
	}
	return nil
}

func (d *GORMDeviceDAO) Delete(ctx context.Context, id string) error {
	result := d.db.WithContext(ctx).Where("id = ?", id).Delete(&Device{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrDeviceNotFound
	}
	return nil
}

func (d *GORMDeviceDAO) DeleteBatch(ctx context.Context, ids []string) (int64, error) {
	result := d.db.WithContext(ctx).Where("id IN ?", ids).Delete(&Device{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (d *GORMDeviceDAO) CountByStatus(ctx context.Context, tenantID string) (int64, int64, int64, int64, error) {
	type statusCount struct {
		Status string
		Count  int64
	}
	var counts []statusCount
	err := d.db.WithContext(ctx).Model(&Device{}).
		Select("status, COUNT(*) as count").
		Where("tenant_id = ?", tenantID).
		Group("status").
		Scan(&counts).Error
	if err != nil {
		return 0, 0, 0, 0, err
	}

	var pending, active, offline, disabled int64
	for _, c := range counts {
		switch c.Status {
		case "pending":
			pending = c.Count
		case "active":
			active = c.Count
		case "offline":
			offline = c.Count
		case "disabled":
			disabled = c.Count
		}
	}
	return pending, active, offline, disabled, nil
}

func (d *GORMDeviceDAO) UpdateStatus(ctx context.Context, id, status string) error {
	result := d.db.WithContext(ctx).Model(&Device{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrDeviceNotFound
	}
	return nil
}

func (d *GORMDeviceDAO) UpdateStatusBatch(ctx context.Context, ids []string, status string) (int64, error) {
	result := d.db.WithContext(ctx).Model(&Device{}).
		Where("id IN ?", ids).
		Update("status", status)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (d *GORMDeviceDAO) UnbindByDeviceIDs(ctx context.Context, ids []string) error {
	return d.db.WithContext(ctx).Model(&Device{}).
		Where("id IN ?", ids).
		Select("parking_lot_id", "gate_id").
		Updates(map[string]interface{}{"parking_lot_id": nil, "gate_id": nil}).Error
}
