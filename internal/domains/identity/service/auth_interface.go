package service

import (
	"context"
	"time"

	"github.com/parkhub/api/internal/domains/identity/domain"
)

type LoginRequest struct {
	Username string
	Password string
}

type SmsLoginRequest struct {
	Phone string
	Code  string
}

type LoginResponse struct {
	TokenPair       domain.TokenPair
	User            *domain.User
	AccessExpiresIn int32
}

type RefreshTokenRequest struct {
	RefreshToken string
}

type RefreshTokenResponse struct {
	TokenPair       domain.TokenPair
	AccessExpiresIn int32
}

type LogoutRequest struct {
	RefreshToken string
}

type SmsCodeVerifier interface {
	VerifyCode(ctx context.Context, phone string, code string) error
}

//go:generate mockgen -source=./auth_interface.go -package=servicemocks -destination=./mocks/auth_service.mock.go AuthService

type AuthService interface {
	Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error)
	SmsLogin(ctx context.Context, req *SmsLoginRequest) (*LoginResponse, error)
	RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error)
	Logout(ctx context.Context, req *LogoutRequest) error
}

func ParseTTL(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 15 * time.Minute
	}
	return d
}
