package repository

import (
	"context"

	"github.com/parkhub/api/internal/domains/identity/domain"
)

type UserFilter struct {
	TenantID string
	Role     domain.UserRole
	Status   domain.UserStatus
	Keyword  string
}

//go:generate mockgen -source=./user_interface.go -package=repomocks -destination=./mocks/user_repo.mock.go UserRepo

type UserRepo interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	List(ctx context.Context, filter UserFilter, page, pageSize int) ([]*domain.User, int64, error)
	Update(ctx context.Context, user *domain.User) error
	BatchCreate(ctx context.Context, users []*domain.User) error
}
