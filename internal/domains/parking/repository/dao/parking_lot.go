package dao

import (
	"context"
	"strings"

	"github.com/parkhub/api/internal/domains/parking/errs"

	"gorm.io/gorm"
)

type ParkingLot struct {
	ID              string `gorm:"primaryKey;type:varchar(36)"`
	TenantID        string `gorm:"type:varchar(36);uniqueIndex:idx_tenant_name;index"`
	Name            string `gorm:"type:varchar(100);uniqueIndex:idx_tenant_name"`
	Address         string `gorm:"type:varchar(255)"`
	TotalSpaces     int    `gorm:"type:int"`
	AvailableSpaces int    `gorm:"type:int"`
	LotType         string `gorm:"type:varchar(20)"`
	Status          string `gorm:"type:varchar(20)"`
	CreatedAt       int64  `gorm:"autoCreateTime:milli"`
	UpdatedAt       int64  `gorm:"autoUpdateTime:milli"`
}

func (ParkingLot) TableName() string {
	return "parking_lots"
}

type ParkingLotFilter struct {
	TenantID string
	Status   string
	LotType  string
	Keyword  string
}

//go:generate mockgen -source=./parking_lot.go -package=daomocks -destination=./mocks/parking_lot.mock.go ParkingLotDAO

type ParkingLotDAO interface {
	Insert(ctx context.Context, lot *ParkingLot) error
	FindByID(ctx context.Context, tenantID, id string) (*ParkingLot, error)
	FindAll(ctx context.Context, filter ParkingLotFilter, page, pageSize int) ([]*ParkingLot, int64, error)
	Update(ctx context.Context, lot *ParkingLot) error
	Delete(ctx context.Context, tenantID, id string) error
	SumStats(ctx context.Context, tenantID string) (totalSpaces, availableSpaces int64, err error)
}

type GORMParkingLotDAO struct {
	db *gorm.DB
}

func NewParkingLotDAO(db *gorm.DB) ParkingLotDAO {
	return &GORMParkingLotDAO{db: db}
}

func (d *GORMParkingLotDAO) Insert(ctx context.Context, lot *ParkingLot) error {
	err := d.db.WithContext(ctx).Create(lot).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return errs.ErrParkingLotNameDuplicate
		}
		return err
	}
	return nil
}

func (d *GORMParkingLotDAO) FindByID(ctx context.Context, tenantID, id string) (*ParkingLot, error) {
	var lot ParkingLot
	err := d.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&lot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrParkingLotNotFound
		}
		return nil, err
	}
	return &lot, nil
}

func (d *GORMParkingLotDAO) FindAll(ctx context.Context, filter ParkingLotFilter, page, pageSize int) ([]*ParkingLot, int64, error) {
	query := d.db.WithContext(ctx).Model(&ParkingLot{})

	query = query.Where("tenant_id = ?", filter.TenantID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.LotType != "" {
		query = query.Where("lot_type = ?", filter.LotType)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR address LIKE ?", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var lots []*ParkingLot
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&lots).Error; err != nil {
		return nil, 0, err
	}

	return lots, total, nil
}

func (d *GORMParkingLotDAO) Update(ctx context.Context, lot *ParkingLot) error {
	result := d.db.WithContext(ctx).Model(&ParkingLot{}).Where("tenant_id = ? AND id = ?", lot.TenantID, lot.ID).Updates(lot)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrParkingLotNotFound
	}
	return nil
}

func (d *GORMParkingLotDAO) Delete(ctx context.Context, tenantID, id string) error {
	result := d.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(&ParkingLot{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrParkingLotNotFound
	}
	return nil
}

func (d *GORMParkingLotDAO) SumStats(ctx context.Context, tenantID string) (int64, int64, error) {
	var result struct {
		TotalSpaces     int64
		AvailableSpaces int64
	}
	err := d.db.WithContext(ctx).Model(&ParkingLot{}).
		Select("COALESCE(SUM(total_spaces), 0) as total_spaces, COALESCE(SUM(available_spaces), 0) as available_spaces").
		Where("tenant_id = ?", tenantID).
		Scan(&result).Error
	if err != nil {
		return 0, 0, err
	}
	return result.TotalSpaces, result.AvailableSpaces, nil
}
