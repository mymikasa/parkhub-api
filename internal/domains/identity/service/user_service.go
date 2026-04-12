package service

import (
	"context"
	"errors"
	"math"

	"github.com/google/uuid"
	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/repository"
	"golang.org/x/crypto/bcrypt"
)

type userService struct {
	userRepo repository.UserRepo
}

func NewUserService(repo repository.UserRepo) UserService {
	return &userService{userRepo: repo}
}

func (s *userService) Create(ctx context.Context, req *CreateUserRequest) (*domain.User, error) {
	if len(req.Password) < 8 {
		return nil, errs.ErrPasswordTooShort
	}
	if req.Role == "" {
		return nil, errs.ErrInvalidRole
	}

	existing, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil && !errors.Is(err, errs.ErrUserNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, errs.ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := domain.NewUser(req.Username, string(hash), req.RealName, req.Role, req.TenantID)
	user.ID = uuid.New().String()
	user.Email = req.Email
	user.Phone = req.Phone

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) GetByID(ctx context.Context, req *GetUserRequest) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, req.UserID)
}

func (s *userService) List(ctx context.Context, req *ListUsersRequest) (*UserListResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filter := repository.UserFilter{
		TenantID: req.TenantID,
		Role:     req.Role,
		Status:   req.Status,
		Keyword:  req.Keyword,
	}

	users, total, err := s.userRepo.List(ctx, filter, page, pageSize)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	return &UserListResponse{
		Users:      users,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *userService) Update(ctx context.Context, req *UpdateUserRequest) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = req.Email
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}
	if req.RealName != nil {
		user.RealName = *req.RealName
	}
	if req.Role != nil {
		user.Role = *req.Role
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) Freeze(ctx context.Context, req *UserActionRequest) error {
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	if err := user.Freeze(); err != nil {
		return err
	}

	return s.userRepo.Update(ctx, user)
}

func (s *userService) Unfreeze(ctx context.Context, req *UserActionRequest) error {
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	if err := user.Unfreeze(); err != nil {
		return err
	}

	return s.userRepo.Update(ctx, user)
}

func (s *userService) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	if len(req.NewPassword) < 8 {
		return errs.ErrPasswordTooShort
	}

	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	return s.userRepo.Update(ctx, user)
}

func (s *userService) UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.RealName != nil {
		user.RealName = *req.RealName
	}
	if req.Email != nil {
		user.Email = req.Email
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) ChangePassword(ctx context.Context, userID string, req *ChangePasswordRequest) error {
	if len(req.NewPassword) < 8 {
		return errs.ErrPasswordTooShort
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return errs.ErrPasswordIncorrect
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	return s.userRepo.Update(ctx, user)
}

func (s *userService) ImportUsers(ctx context.Context, req *ImportUsersRequest) (*ImportResult, error) {
	result := &ImportResult{
		Total: len(req.Users),
	}

	for i, item := range req.Users {
		_, err := s.Create(ctx, &CreateUserRequest{
			TenantID: item.TenantID,
			Username: item.Username,
			Password: item.Password,
			Email:    item.Email,
			Phone:    item.Phone,
			RealName: item.RealName,
			Role:     item.Role,
		})
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Index:   i,
				Message: err.Error(),
			})
		} else {
			result.Success++
		}
	}

	return result, nil
}
