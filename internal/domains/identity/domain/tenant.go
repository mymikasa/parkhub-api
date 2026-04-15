package domain

import (
	"github.com/parkhub/api/internal/domains/identity/errs"
)

type TenantStatus string

const (
	TenantStatusActive TenantStatus = "active" // 正常运营
	TenantStatusFrozen TenantStatus = "frozen" // 已冻结
)

type PlanType string

const (
	PlanTypeFree       PlanType = "free"
	PlanTypeBasic      PlanType = "basic"
	PlanTypePro        PlanType = "pro"
	PlanTypeEnterprise PlanType = "enterprise"
)

type Tenant struct {
	ID              string
	Name            string
	ContactName     string
	ContactPhone    string
	ContactEmail    string
	Status          TenantStatus
	Address         string
	PlanType        PlanType
	CreatedAt       int64
	UpdatedAt       int64
	Description     string
	CreditCode      string
	Remark          string
	ParkingLotCount int32
}

func NewTenant(name, contactName, contactPhone, contactEmail, address string, planType PlanType) *Tenant {
	return &Tenant{
		Name:         name,
		ContactName:  contactName,
		ContactPhone: contactPhone,
		ContactEmail: contactEmail,
		Address:      address,
		Status:       TenantStatusActive,
		PlanType:     planType,
	}
}

func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

func (t *Tenant) Freeze() error {
	if t.Status != TenantStatusActive {
		return errs.ErrTenantInvalidStatus
	}
	t.Status = TenantStatusFrozen
	return nil
}

func (t *Tenant) Unfreeze() error {
	if t.Status != TenantStatusFrozen {
		return errs.ErrTenantInvalidStatus
	}
	t.Status = TenantStatusActive
	return nil
}
