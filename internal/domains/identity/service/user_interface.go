package service

import (
	"context"

	"github.com/parkhub/api/internal/domains/identity/domain"
)

type CreateUserRequest struct {
	TenantID *string
	Username string
	Password string
	Email    *string
	Phone    *string
	RealName string
	Role     domain.UserRole
}

type GetUserRequest struct {
	UserID string
}

type ListUsersRequest struct {
	TenantID string
	Role     domain.UserRole
	Status   domain.UserStatus
	Keyword  string
	Page     int
	PageSize int
}

type UserListResponse struct {
	Users      []*domain.User
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

type UpdateUserRequest struct {
	UserID   string
	Username *string
	Email    *string
	Phone    *string
	RealName *string
	Role     *domain.UserRole
}

type UserActionRequest struct {
	UserID string
}

type ResetPasswordRequest struct {
	UserID      string
	NewPassword string
}

type UpdateProfileRequest struct {
	RealName *string
	Email    *string
	Phone    *string
}

type ChangePasswordRequest struct {
	OldPassword string
	NewPassword string
}

type ImportUserItem struct {
	TenantID *string
	Username string
	Password string
	Email    *string
	Phone    *string
	RealName string
	Role     domain.UserRole
}

type ImportUsersRequest struct {
	Users []ImportUserItem
}

type ImportResult struct {
	Total   int
	Success int
	Failed  int
	Errors  []ImportError
}

type ImportError struct {
	Index   int
	Message string
}

//go:generate mockgen -source=./user_interface.go -package=servicemocks -destination=./mocks/user_service.mock.go UserService

type UserService interface {
	Create(ctx context.Context, req *CreateUserRequest) (*domain.User, error)
	GetByID(ctx context.Context, req *GetUserRequest) (*domain.User, error)
	List(ctx context.Context, req *ListUsersRequest) (*UserListResponse, error)
	Update(ctx context.Context, req *UpdateUserRequest) (*domain.User, error)
	Freeze(ctx context.Context, req *UserActionRequest) error
	Unfreeze(ctx context.Context, req *UserActionRequest) error
	ResetPassword(ctx context.Context, req *ResetPasswordRequest) error
	UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*domain.User, error)
	ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error
	ImportUsers(ctx context.Context, req *ImportUsersRequest) (*ImportResult, error)
}
