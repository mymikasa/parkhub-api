package dao

import (
	"context"

	"gorm.io/gorm"
)

type SmsRecord struct {
	ID          string  `gorm:"primaryKey;type:varchar(36)"`
	Phone       string  `gorm:"type:varchar(20);index"`
	Purpose     string  `gorm:"type:varchar(32);index"`
	Code        string  `gorm:"type:varchar(10)"`
	Status      string  `gorm:"type:varchar(20);index"`
	ProviderErr *string `gorm:"type:text"`
	CreatedAt   int64   `gorm:"autoCreateTime:milli;index"`
}

func (SmsRecord) TableName() string {
	return "sms_records"
}

type SmsRecordFilter struct {
	Phone     string
	Purpose   string
	Status    string
	StartTime *int64
	EndTime   *int64
}

type SmsRecordDAO interface {
	Insert(ctx context.Context, record *SmsRecord) error
	FindByID(ctx context.Context, id string) (*SmsRecord, error)
	FindByPhone(ctx context.Context, phone string, page, pageSize int) ([]*SmsRecord, int64, error)
	List(ctx context.Context, filter SmsRecordFilter, page, pageSize int) ([]*SmsRecord, int64, error)
}

type GORMSmsRecordDAO struct {
	db *gorm.DB
}

func NewSmsRecordDAO(db *gorm.DB) SmsRecordDAO {
	return &GORMSmsRecordDAO{db: db}
}

func (d *GORMSmsRecordDAO) Insert(ctx context.Context, record *SmsRecord) error {
	return d.db.WithContext(ctx).Create(record).Error
}

func (d *GORMSmsRecordDAO) FindByID(ctx context.Context, id string) (*SmsRecord, error) {
	var record SmsRecord
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (d *GORMSmsRecordDAO) FindByPhone(ctx context.Context, phone string, page, pageSize int) ([]*SmsRecord, int64, error) {
	query := d.db.WithContext(ctx).Model(&SmsRecord{}).Where("phone = ?", phone)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []*SmsRecord
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (d *GORMSmsRecordDAO) List(ctx context.Context, filter SmsRecordFilter, page, pageSize int) ([]*SmsRecord, int64, error) {
	query := d.db.WithContext(ctx).Model(&SmsRecord{})

	if filter.Phone != "" {
		query = query.Where("phone = ?", filter.Phone)
	}
	if filter.Purpose != "" {
		query = query.Where("purpose = ?", filter.Purpose)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []*SmsRecord
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}
