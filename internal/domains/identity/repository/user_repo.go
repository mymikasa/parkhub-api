package repository

import (
	"context"
	"time"

	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/repository/dao"
)

type userRepo struct {
	dao dao.UserDAO
}

func NewUserRepo(d dao.UserDAO) UserRepo {
	return &userRepo{dao: d}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	return r.dao.Insert(ctx, toUserEntity(user))
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	d, err := r.dao.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDomainUser(d), nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	d, err := r.dao.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return toDomainUser(d), nil
}

func (r *userRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	d, err := r.dao.FindByPhone(ctx, phone)
	if err != nil {
		return nil, err
	}
	return toDomainUser(d), nil
}

func (r *userRepo) List(ctx context.Context, filter UserFilter, page, pageSize int) ([]*domain.User, int64, error) {
	daoFilter := dao.UserFilter{
		TenantID: filter.TenantID,
		Role:     string(filter.Role),
		Status:   string(filter.Status),
		Keyword:  filter.Keyword,
	}
	records, total, err := r.dao.FindAll(ctx, daoFilter, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	users := make([]*domain.User, 0, len(records))
	for _, rec := range records {
		users = append(users, toDomainUser(rec))
	}
	return users, total, nil
}

func (r *userRepo) Update(ctx context.Context, user *domain.User) error {
	return r.dao.Update(ctx, toUserEntity(user))
}

func (r *userRepo) BatchCreate(ctx context.Context, users []*domain.User) error {
	entities := make([]*dao.User, 0, len(users))
	for _, u := range users {
		entities = append(entities, toUserEntity(u))
	}
	return r.dao.BatchInsert(ctx, entities)
}

func toDomainUser(d *dao.User) *domain.User {
	if d == nil {
		return nil
	}
	u := &domain.User{
		ID:           d.ID,
		TenantID:     d.TenantID,
		Username:     d.Username,
		Email:        d.Email,
		Phone:        d.Phone,
		PasswordHash: d.PasswordHash,
		RealName:     d.RealName,
		Role:         domain.UserRole(d.Role),
		Status:       domain.UserStatus(d.Status),
		CreatedAt:    time.UnixMilli(d.CreatedAt),
		UpdatedAt:    time.UnixMilli(d.UpdatedAt),
	}
	if d.LastLoginAt != nil {
		t := time.UnixMilli(*d.LastLoginAt)
		u.LastLoginAt = &t
	}
	return u
}

func toUserEntity(u *domain.User) *dao.User {
	if u == nil {
		return nil
	}
	d := &dao.User{
		ID:           u.ID,
		TenantID:     u.TenantID,
		Username:     u.Username,
		Email:        u.Email,
		Phone:        u.Phone,
		PasswordHash: u.PasswordHash,
		RealName:     u.RealName,
		Role:         string(u.Role),
		Status:       string(u.Status),
	}
	if !u.CreatedAt.IsZero() {
		d.CreatedAt = u.CreatedAt.UnixMilli()
	}
	if !u.UpdatedAt.IsZero() {
		d.UpdatedAt = u.UpdatedAt.UnixMilli()
	}
	if u.LastLoginAt != nil {
		millis := u.LastLoginAt.UnixMilli()
		d.LastLoginAt = &millis
	}
	return d
}
