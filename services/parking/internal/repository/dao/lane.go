package dao

import (
	"context"
	"strings"

	"github.com/parkhub/api/services/parking/internal/errs"

	"gorm.io/gorm"
)

type Lane struct {
	ID           string  `gorm:"primaryKey;type:varchar(36)"`
	TenantID     string  `gorm:"type:varchar(36);uniqueIndex:idx_lane_lot_name;index"`
	ParkingLotID string  `gorm:"type:varchar(36);uniqueIndex:idx_lane_lot_name;index"`
	Name         string  `gorm:"type:varchar(100);uniqueIndex:idx_lane_lot_name"`
	LaneType     string  `gorm:"type:varchar(20)"`
	DeviceID     *string `gorm:"type:varchar(36)"`
	CreatedAt    int64   `gorm:"autoCreateTime:milli"`
	UpdatedAt    int64   `gorm:"autoUpdateTime:milli"`
}

func (Lane) TableName() string { return "lanes" }

type LaneTypeCount struct {
	Entry int
	Exit  int
}

type LaneDAO interface {
	Insert(ctx context.Context, lane *Lane) error
	FindByID(ctx context.Context, tenantID, id string) (*Lane, error)
	FindByParkingLotID(ctx context.Context, tenantID, parkingLotID string) ([]*Lane, error)
	Update(ctx context.Context, lane *Lane) error
	Delete(ctx context.Context, tenantID, id string) error
	ExistsByName(ctx context.Context, parkingLotID, name string) (bool, error)
	CountByParkingLots(ctx context.Context, tenantID string, parkingLotIDs []string) (map[string]*LaneTypeCount, error)
}

type GORMLaneDAO struct {
	db *gorm.DB
}

func NewLaneDAO(db *gorm.DB) LaneDAO {
	return &GORMLaneDAO{db: db}
}

func (d *GORMLaneDAO) Insert(ctx context.Context, lane *Lane) error {
	err := d.db.WithContext(ctx).Create(lane).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return errs.ErrLaneNameDuplicate
		}
		return err
	}
	return nil
}

func (d *GORMLaneDAO) FindByID(ctx context.Context, tenantID, id string) (*Lane, error) {
	var lane Lane
	err := d.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&lane).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrLaneNotFound
		}
		return nil, err
	}
	return &lane, nil
}

func (d *GORMLaneDAO) FindByParkingLotID(ctx context.Context, tenantID, parkingLotID string) ([]*Lane, error) {
	var lanes []*Lane
	err := d.db.WithContext(ctx).
		Where("tenant_id = ? AND parking_lot_id = ?", tenantID, parkingLotID).
		Order("created_at ASC").
		Find(&lanes).Error
	if err != nil {
		return nil, err
	}
	return lanes, nil
}

func (d *GORMLaneDAO) Update(ctx context.Context, lane *Lane) error {
	result := d.db.WithContext(ctx).Model(&Lane{}).
		Where("tenant_id = ? AND id = ?", lane.TenantID, lane.ID).
		Select("name", "lane_type", "device_id", "updated_at").
		Updates(lane)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrLaneNotFound
	}
	return nil
}

func (d *GORMLaneDAO) Delete(ctx context.Context, tenantID, id string) error {
	result := d.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		Delete(&Lane{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrLaneNotFound
	}
	return nil
}

func (d *GORMLaneDAO) ExistsByName(ctx context.Context, parkingLotID, name string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&Lane{}).
		Where("parking_lot_id = ? AND name = ?", parkingLotID, name).
		Count(&count).Error
	return count > 0, err
}

func (d *GORMLaneDAO) CountByParkingLots(ctx context.Context, tenantID string, parkingLotIDs []string) (map[string]*LaneTypeCount, error) {
	if len(parkingLotIDs) == 0 {
		return map[string]*LaneTypeCount{}, nil
	}

	type row struct {
		ParkingLotID string
		LaneType     string
		Count        int
	}
	var results []row
	err := d.db.WithContext(ctx).Model(&Lane{}).
		Select("parking_lot_id, lane_type, count(*) as count").
		Where("tenant_id = ? AND parking_lot_id IN ?", tenantID, parkingLotIDs).
		Group("parking_lot_id, lane_type").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]*LaneTypeCount)
	for _, r := range results {
		c, ok := counts[r.ParkingLotID]
		if !ok {
			c = &LaneTypeCount{}
			counts[r.ParkingLotID] = c
		}
		switch r.LaneType {
		case "entry":
			c.Entry = r.Count
		case "exit":
			c.Exit = r.Count
		}
	}
	return counts, nil
}
