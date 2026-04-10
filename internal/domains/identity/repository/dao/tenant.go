package dao

import (
	"context"
	"strings"

	"github.com/parkhub/api/internal/domains/identity/errs"

	"gorm.io/gorm"
)

type Tenant struct {
	ID           string `gorm:"primaryKey;type:varchar(36)"`
	Name         string `gorm:"type:varchar(100);uniqueIndex"`
	ContactName  string `gorm:"type:varchar(100)"`
	ContactPhone string `gorm:"type:varchar(20)"`
	ContactEmail string `gorm:"type:varchar(100)"`
	Status       string `gorm:"type:varchar(20)"`
	Address      string `gorm:"type:varchar(255)"`
	PlanType     string `gorm:"type:varchar(20)"`
	CreatedAt    int64  `gorm:"autoCreateTime:milli"`
	UpdatedAt    int64  `gorm:"autoUpdateTime:milli"`
}

func (Tenant) TableName() string {
	return "tenants"
}

type TenantFilter struct {
	Status  string
	Keyword string
}

//go:generate mockgen -source=./tenant.go -package=daomocks -destination=./mocks/tenant.mock.go TenantDAO

type TenantDAO interface {
	Insert(ctx context.Context, tenant *Tenant) error
	FindByID(ctx context.Context, id string) (*Tenant, error)
	FindByName(ctx context.Context, name string) (*Tenant, error)
	FindAll(ctx context.Context, filter TenantFilter, page, pageSize int) ([]*Tenant, int64, error)
	Update(ctx context.Context, tenant *Tenant) error
	Delete(ctx context.Context, id string) error
}

type GORMTenantDAO struct {
	db *gorm.DB
}

func NewTenantDAO(db *gorm.DB) TenantDAO {
	return &GORMTenantDAO{db: db}
}

func (d *GORMTenantDAO) Insert(ctx context.Context, tenant *Tenant) error {
	err := d.db.WithContext(ctx).Create(tenant).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return errs.ErrTenantAlreadyExists
		}
		return err
	}
	return nil
}

func (d *GORMTenantDAO) FindByID(ctx context.Context, id string) (*Tenant, error) {
	var tenant Tenant
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&tenant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

func (d *GORMTenantDAO) FindByName(ctx context.Context, name string) (*Tenant, error) {
	var tenant Tenant
	err := d.db.WithContext(ctx).Where("name = ?", name).First(&tenant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

func (d *GORMTenantDAO) FindAll(ctx context.Context, filter TenantFilter, page, pageSize int) ([]*Tenant, int64, error) {
	query := d.db.WithContext(ctx).Model(&Tenant{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR contact_name LIKE ? OR contact_email LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tenants []*Tenant
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tenants).Error; err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

func (d *GORMTenantDAO) Update(ctx context.Context, tenant *Tenant) error {
	result := d.db.WithContext(ctx).Model(&Tenant{}).Where("id = ?", tenant.ID).Updates(tenant)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrTenantNotFound
	}
	return nil
}

func (d *GORMTenantDAO) Delete(ctx context.Context, id string) error {
	result := d.db.WithContext(ctx).Where("id = ?", id).Delete(&Tenant{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrTenantNotFound
	}
	return nil
}
