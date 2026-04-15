package grpc

import (
	"context"

	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/service"
	commonv1 "github.com/parkhub/api/internal/gen/api/proto/common/v1"
	identityv1 "github.com/parkhub/api/internal/gen/api/proto/identity/v1"
	"github.com/parkhub/api/pkg/grpcutil"
	"github.com/parkhub/api/pkg/identityctx"
	"google.golang.org/grpc/codes"
)

type UserGRPCServer struct {
	identityv1.UnimplementedUserServiceServer
	userSvc service.UserService
}

func NewUserGRPCServer(svc service.UserService) *UserGRPCServer {
	return &UserGRPCServer{userSvc: svc}
}

var userErrorMappings = []grpcutil.ErrorMapping{
	{Target: errs.ErrUserNotFound, Code: codes.NotFound},
	{Target: errs.ErrUserAlreadyExists, Code: codes.AlreadyExists},
	{Target: errs.ErrUserInvalidStatus, Code: codes.InvalidArgument},
	{Target: errs.ErrUserFrozen, Code: codes.FailedPrecondition},
	{Target: errs.ErrPasswordIncorrect, Code: codes.InvalidArgument},
	{Target: errs.ErrPasswordTooShort, Code: codes.InvalidArgument},
	{Target: errs.ErrInvalidRole, Code: codes.InvalidArgument},
}

func toUserGRPCError(err error) error {
	return grpcutil.ToGRPCError(err, userErrorMappings)
}

func (s *UserGRPCServer) CreateUser(ctx context.Context, req *identityv1.CreateUserRequest) (*identityv1.CreateUserResponse, error) {
	role := domainRoleFromProto(req.Role)
	if role == "" {
		return nil, toUserGRPCError(errs.ErrInvalidRole)
	}

	user, err := s.userSvc.Create(ctx, &service.CreateUserRequest{
		TenantID: req.TenantId,
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Phone:    req.Phone,
		RealName: req.RealName,
		Role:     role,
	})
	if err != nil {
		return nil, toUserGRPCError(err)
	}
	return &identityv1.CreateUserResponse{User: toProtoUser(user)}, nil
}

func (s *UserGRPCServer) GetUser(ctx context.Context, req *identityv1.GetUserRequest) (*identityv1.GetUserResponse, error) {
	user, err := s.userSvc.GetByID(ctx, &service.GetUserRequest{UserID: req.UserId})
	if err != nil {
		return nil, toUserGRPCError(err)
	}
	return &identityv1.GetUserResponse{User: toProtoUser(user)}, nil
}

func (s *UserGRPCServer) GetCurrentUser(ctx context.Context, _ *identityv1.GetCurrentUserRequest) (*identityv1.GetCurrentUserResponse, error) {
	userID := identityctx.UserID(ctx)
	if userID == "" {
		return nil, grpcutil.ToGRPCError(errs.ErrUserNotFound, userErrorMappings)
	}
	user, err := s.userSvc.GetByID(ctx, &service.GetUserRequest{UserID: userID})
	if err != nil {
		return nil, toUserGRPCError(err)
	}
	return &identityv1.GetCurrentUserResponse{User: toProtoUser(user)}, nil
}

func (s *UserGRPCServer) ListUsers(ctx context.Context, req *identityv1.ListUsersRequest) (*identityv1.ListUsersResponse, error) {
	page := int32(1)
	pageSize := int32(20)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PageSize > 0 {
			pageSize = req.Pagination.PageSize
		}
	}

	resp, err := s.userSvc.List(ctx, &service.ListUsersRequest{
		TenantID: req.TenantId,
		Role:     domainRoleFromProto(req.Role),
		Status:   domainUserStatusFromProto(req.Status),
		Keyword:  req.Keyword,
		Page:     int(page),
		PageSize: int(pageSize),
	})
	if err != nil {
		return nil, toUserGRPCError(err)
	}

	users := make([]*identityv1.User, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, toProtoUser(u))
	}

	return &identityv1.ListUsersResponse{
		Users: users,
		Pagination: &commonv1.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			Total:      resp.Total,
			TotalPages: int32(resp.TotalPages),
		},
	}, nil
}

func (s *UserGRPCServer) UpdateUser(ctx context.Context, req *identityv1.UpdateUserRequest) (*identityv1.UpdateUserResponse, error) {
	svcReq := &service.UpdateUserRequest{
		UserID: req.UserId,
	}
	if req.Username != nil {
		svcReq.Username = req.Username
	}
	if req.Email != nil {
		svcReq.Email = req.Email
	}
	if req.Phone != nil {
		svcReq.Phone = req.Phone
	}
	if req.RealName != nil {
		svcReq.RealName = req.RealName
	}
	if req.Role != nil {
		role := domainRoleFromProto(*req.Role)
		svcReq.Role = &role
	}

	user, err := s.userSvc.Update(ctx, svcReq)
	if err != nil {
		return nil, toUserGRPCError(err)
	}
	return &identityv1.UpdateUserResponse{User: toProtoUser(user)}, nil
}

func (s *UserGRPCServer) FreezeUser(ctx context.Context, req *identityv1.FreezeUserRequest) (*identityv1.FreezeUserResponse, error) {
	if err := s.userSvc.Freeze(ctx, &service.UserActionRequest{UserID: req.UserId}); err != nil {
		return nil, toUserGRPCError(err)
	}
	return &identityv1.FreezeUserResponse{}, nil
}

func (s *UserGRPCServer) UnfreezeUser(ctx context.Context, req *identityv1.UnfreezeUserRequest) (*identityv1.UnfreezeUserResponse, error) {
	if err := s.userSvc.Unfreeze(ctx, &service.UserActionRequest{UserID: req.UserId}); err != nil {
		return nil, toUserGRPCError(err)
	}
	return &identityv1.UnfreezeUserResponse{}, nil
}

func (s *UserGRPCServer) ResetPassword(ctx context.Context, req *identityv1.ResetPasswordRequest) (*identityv1.ResetPasswordResponse, error) {
	if err := s.userSvc.ResetPassword(ctx, &service.ResetPasswordRequest{
		UserID:      req.UserId,
		NewPassword: req.NewPassword,
	}); err != nil {
		return nil, toUserGRPCError(err)
	}
	return &identityv1.ResetPasswordResponse{}, nil
}

func (s *UserGRPCServer) UpdateProfile(ctx context.Context, req *identityv1.UpdateProfileRequest) (*identityv1.UpdateProfileResponse, error) {
	user, err := s.userSvc.UpdateProfile(ctx, req.UserId, &service.UpdateProfileRequest{
		RealName: req.RealName,
		Email:    req.Email,
		Phone:    req.Phone,
	})
	if err != nil {
		return nil, toUserGRPCError(err)
	}
	return &identityv1.UpdateProfileResponse{User: toProtoUser(user)}, nil
}

func (s *UserGRPCServer) ChangePassword(ctx context.Context, req *identityv1.ChangePasswordRequest) (*identityv1.ChangePasswordResponse, error) {
	if err := s.userSvc.ChangePassword(ctx, req.UserId, &service.ChangePasswordRequest{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}); err != nil {
		return nil, toUserGRPCError(err)
	}
	return &identityv1.ChangePasswordResponse{}, nil
}

func (s *UserGRPCServer) ImportUsers(ctx context.Context, req *identityv1.ImportUsersRequest) (*identityv1.ImportUsersResponse, error) {
	items := make([]service.ImportUserItem, 0, len(req.Users))
	for _, u := range req.Users {
		items = append(items, service.ImportUserItem{
			TenantID: u.TenantId,
			Username: u.Username,
			Password: u.Password,
			Email:    u.Email,
			Phone:    u.Phone,
			RealName: u.RealName,
			Role:     domainRoleFromProto(u.Role),
		})
	}

	result, err := s.userSvc.ImportUsers(ctx, &service.ImportUsersRequest{Users: items})
	if err != nil {
		return nil, toUserGRPCError(err)
	}

	importErrors := make([]*identityv1.ImportError, 0, len(result.Errors))
	for _, e := range result.Errors {
		importErrors = append(importErrors, &identityv1.ImportError{
			Index:   int32(e.Index),
			Message: e.Message,
		})
	}

	return &identityv1.ImportUsersResponse{
		Total:   int32(result.Total),
		Success: int32(result.Success),
		Failed:  int32(result.Failed),
		Errors:  importErrors,
	}, nil
}
