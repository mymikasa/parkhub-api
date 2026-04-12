package service

import (
	"context"
	"testing"

	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/repository"
	repomocks "github.com/parkhub/api/internal/domains/identity/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func setupUserService(t *testing.T) (UserService, *repomocks.MockUserRepo) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockRepo := repomocks.NewMockUserRepo(ctrl)
	svc := NewUserService(mockRepo)
	return svc, mockRepo
}

func TestUserService_Create_Success(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetByUsername(gomock.Any(), "alice").Return(nil, errs.ErrUserNotFound)
	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	tenantID := "tenant-1"
	user, err := svc.Create(ctx, &CreateUserRequest{
		TenantID: &tenantID,
		Username: "alice",
		Password: "password123",
		RealName: "Alice",
		Role:     domain.RoleTenantAdmin,
	})

	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, domain.UserStatusActive, user.Status)

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123"))
	assert.NoError(t, err)
}

func TestUserService_Create_DuplicateUsername(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetByUsername(gomock.Any(), "alice").Return(&domain.User{Username: "alice"}, nil)

	_, err := svc.Create(ctx, &CreateUserRequest{
		Username: "alice",
		Password: "password123",
		RealName: "Alice",
		Role:     domain.RoleOperator,
	})

	assert.ErrorIs(t, err, errs.ErrUserAlreadyExists)
}

func TestUserService_GetByID(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&domain.User{
		ID:       "user-1",
		Username: "alice",
	}, nil)

	user, err := svc.GetByID(ctx, &GetUserRequest{UserID: "user-1"})
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
}

func TestUserService_List(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().List(gomock.Any(), repository.UserFilter{TenantID: "t1"}, 1, 20).
		Return([]*domain.User{
			{ID: "u1", Username: "alice"},
			{ID: "u2", Username: "bob"},
		}, int64(2), nil)

	resp, err := svc.List(ctx, &ListUsersRequest{TenantID: "t1"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)
	assert.Equal(t, 1, resp.TotalPages)
}

func TestUserService_List_PageSizeCap(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().List(gomock.Any(), gomock.Any(), 1, 100).
		Return([]*domain.User{}, int64(0), nil)

	_, err := svc.List(ctx, &ListUsersRequest{PageSize: 200})
	assert.NoError(t, err)
}

func TestUserService_Update(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&domain.User{
		ID:       "user-1",
		Username: "alice",
		RealName: "Alice",
		Role:     domain.RoleOperator,
		Status:   domain.UserStatusActive,
	}, nil)
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	newName := "Alice Updated"
	user, err := svc.Update(ctx, &UpdateUserRequest{
		UserID:   "user-1",
		RealName: &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", user.RealName)
}

func TestUserService_Freeze(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&domain.User{
		ID:     "user-1",
		Status: domain.UserStatusActive,
	}, nil)
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	err := svc.Freeze(ctx, &UserActionRequest{UserID: "user-1"})
	assert.NoError(t, err)
}

func TestUserService_Freeze_AlreadyFrozen(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&domain.User{
		ID:     "user-1",
		Status: domain.UserStatusFrozen,
	}, nil)

	err := svc.Freeze(ctx, &UserActionRequest{UserID: "user-1"})
	assert.ErrorIs(t, err, errs.ErrUserInvalidStatus)
}

func TestUserService_Unfreeze(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&domain.User{
		ID:     "user-1",
		Status: domain.UserStatusFrozen,
	}, nil)
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	err := svc.Unfreeze(ctx, &UserActionRequest{UserID: "user-1"})
	assert.NoError(t, err)
}

func TestUserService_ResetPassword(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&domain.User{
		ID:           "user-1",
		PasswordHash: "oldhash",
	}, nil)
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, user *domain.User) error {
		err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("newpass123"))
		assert.NoError(t, err)
		return nil
	})

	err := svc.ResetPassword(ctx, &ResetPasswordRequest{
		UserID:      "user-1",
		NewPassword: "newpass123",
	})
	assert.NoError(t, err)
}

func TestUserService_ChangePassword_Success(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	oldHash, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)
	mockRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&domain.User{
		ID:           "user-1",
		PasswordHash: string(oldHash),
	}, nil)
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	err := svc.ChangePassword(ctx, "user-1", &ChangePasswordRequest{
		OldPassword: "oldpass",
		NewPassword: "newpass",
	})
	assert.NoError(t, err)
}

func TestUserService_ChangePassword_WrongOldPassword(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	oldHash, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)
	mockRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&domain.User{
		ID:           "user-1",
		PasswordHash: string(oldHash),
	}, nil)

	err := svc.ChangePassword(ctx, "user-1", &ChangePasswordRequest{
		OldPassword: "wrongpass",
		NewPassword: "newpass",
	})
	assert.ErrorIs(t, err, errs.ErrPasswordIncorrect)
}

func TestUserService_UpdateProfile(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(&domain.User{
		ID:       "user-1",
		RealName: "Alice",
		Status:   domain.UserStatusActive,
	}, nil)
	mockRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	newName := "Alice Updated"
	email := "alice@new.com"
	user, err := svc.UpdateProfile(ctx, "user-1", &UpdateProfileRequest{
		RealName: &newName,
		Email:    &email,
	})

	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", user.RealName)
	assert.Equal(t, &email, user.Email)
}

func TestUserService_ImportUsers(t *testing.T) {
	svc, mockRepo := setupUserService(t)
	ctx := context.Background()

	tenantID := "t1"

	// First user: success
	mockRepo.EXPECT().GetByUsername(gomock.Any(), "alice").Return(nil, errs.ErrUserNotFound)
	mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	// Second user: duplicate
	mockRepo.EXPECT().GetByUsername(gomock.Any(), "bob").Return(&domain.User{Username: "bob"}, nil)

	result, err := svc.ImportUsers(ctx, &ImportUsersRequest{
		Users: []ImportUserItem{
			{TenantID: &tenantID, Username: "alice", Password: "pass1", RealName: "Alice", Role: domain.RoleOperator},
			{TenantID: &tenantID, Username: "bob", Password: "pass2", RealName: "Bob", Role: domain.RoleOperator},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 1, result.Success)
	assert.Equal(t, 1, result.Failed)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, 1, result.Errors[0].Index)
}
