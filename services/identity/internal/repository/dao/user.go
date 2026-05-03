package dao

import (
	"context"
	"strings"

	"github.com/parkhub/api/services/identity/internal/errs"

	"gorm.io/gorm"
)

type User struct {
	ID           string  `gorm:"primaryKey;type:varchar(36)"`
	TenantID     *string `gorm:"type:varchar(36);uniqueIndex:idx_tenant_username"`
	Username     string  `gorm:"type:varchar(100);uniqueIndex:idx_tenant_username"`
	Email        *string `gorm:"type:varchar(100)"`
	Phone        *string `gorm:"type:varchar(20);uniqueIndex:idx_phone"`
	PasswordHash string  `gorm:"type:varchar(255)"`
	RealName     string  `gorm:"type:varchar(100)"`
	Role         string  `gorm:"type:varchar(20)"`
	Status       string  `gorm:"type:varchar(20)"`
	LastLoginAt  *int64  `gorm:"type:bigint"`
	CreatedAt    int64   `gorm:"autoCreateTime:milli"`
	UpdatedAt    int64   `gorm:"autoUpdateTime:milli"`
}

func (User) TableName() string {
	return "users"
}

type UserFilter struct {
	TenantID string
	Role     string
	Status   string
	Keyword  string
}

type UserDAO interface {
	Insert(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	FindAll(ctx context.Context, filter UserFilter, page, pageSize int) ([]*User, int64, error)
	Update(ctx context.Context, user *User) error
	BatchInsert(ctx context.Context, users []*User) error
}

type GORMUserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) UserDAO {
	return &GORMUserDAO{db: db}
}

func (d *GORMUserDAO) Insert(ctx context.Context, user *User) error {
	err := d.db.WithContext(ctx).Create(user).Error
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE") {
			return errs.ErrUserAlreadyExists
		}
		return err
	}
	return nil
}

func (d *GORMUserDAO) FindByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (d *GORMUserDAO) FindByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := d.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (d *GORMUserDAO) FindByPhone(ctx context.Context, phone string) (*User, error) {
	var user User
	err := d.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (d *GORMUserDAO) FindAll(ctx context.Context, filter UserFilter, page, pageSize int) ([]*User, int64, error) {
	query := d.db.WithContext(ctx).Model(&User{})

	if filter.TenantID != "" {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("username LIKE ? OR real_name LIKE ? OR email LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []*User
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (d *GORMUserDAO) Update(ctx context.Context, user *User) error {
	result := d.db.WithContext(ctx).Model(&User{}).Where("id = ?", user.ID).Updates(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errs.ErrUserNotFound
	}
	return nil
}

func (d *GORMUserDAO) BatchInsert(ctx context.Context, users []*User) error {
	return d.db.WithContext(ctx).Create(users).Error
}
