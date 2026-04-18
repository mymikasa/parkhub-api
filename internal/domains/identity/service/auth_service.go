package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/parkhub/api/internal/config"
	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/repository"
	smserrs "github.com/parkhub/api/internal/domains/sms/errs"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo    repository.UserRepo
	refreshRepo repository.RefreshTokenRepo
	signer      domain.TokenSigner
	smsVerifier SmsCodeVerifier
	cfg         config.AuthConfig
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

func NewAuthService(
	userRepo repository.UserRepo,
	refreshRepo repository.RefreshTokenRepo,
	signer domain.TokenSigner,
	cfg config.AuthConfig,
	smsVerifier SmsCodeVerifier,
) AuthService {
	return &authService{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		signer:      signer,
		smsVerifier: smsVerifier,
		cfg:         cfg,
		accessTTL:   ParseTTL(cfg.AccessTTL),
		refreshTTL:  ParseTTL(cfg.RefreshTTL),
	}
}

func (s *authService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			return nil, errs.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errs.ErrInvalidCredentials
	}

	if !user.IsActive() {
		return nil, errs.ErrUserFrozen
	}

	pair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user.LastLoginAt = &now
	_ = s.userRepo.Update(ctx, user)

	return &LoginResponse{
		TokenPair:       *pair,
		User:            user,
		AccessExpiresIn: int32(s.accessTTL.Seconds()),
	}, nil
}

func (s *authService) SmsLogin(ctx context.Context, req *SmsLoginRequest) (*LoginResponse, error) {
	if err := s.smsVerifier.VerifyCode(ctx, req.Phone, req.Code); err != nil {
		if isSmsCredentialError(err) {
			return nil, errs.ErrInvalidCredentials
		}
		return nil, err
	}

	user, err := s.userRepo.GetByPhone(ctx, req.Phone)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			return nil, errs.ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive() {
		return nil, errs.ErrUserFrozen
	}

	pair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user.LastLoginAt = &now
	_ = s.userRepo.Update(ctx, user)

	return &LoginResponse{
		TokenPair:       *pair,
		User:            user,
		AccessExpiresIn: int32(s.accessTTL.Seconds()),
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*RefreshTokenResponse, error) {
	claims, err := s.signer.Verify(req.RefreshToken)
	if err != nil {
		return nil, errs.ErrRefreshTokenInvalid
	}

	if claims.TokenUse != domain.TokenUseRefresh {
		return nil, errs.ErrRefreshTokenInvalid
	}

	_, ok, err := s.refreshRepo.Consume(ctx, claims.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.ErrRefreshTokenRevoked
	}

	userID := claims.Subject
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive() {
		return nil, errs.ErrUserFrozen
	}

	pair, err := s.generateTokenPair(ctx, user)
	if err != nil {
		return nil, err
	}

	return &RefreshTokenResponse{
		TokenPair:       *pair,
		AccessExpiresIn: int32(s.accessTTL.Seconds()),
	}, nil
}

func (s *authService) Logout(ctx context.Context, req *LogoutRequest) error {
	claims, err := s.signer.Verify(req.RefreshToken)
	if err != nil {
		return errs.ErrRefreshTokenInvalid
	}

	if claims.TokenUse != domain.TokenUseRefresh {
		return errs.ErrRefreshTokenInvalid
	}

	return s.refreshRepo.Revoke(ctx, claims.ID)
}

func (s *authService) generateTokenPair(ctx context.Context, user *domain.User) (*domain.TokenPair, error) {
	now := time.Now()
	accessExpiresAt := now.Add(s.accessTTL)
	refreshExpiresAt := now.Add(s.refreshTTL)

	accessClaims := domain.Claims{
		TokenUse: domain.TokenUseAccess,
		UserID:   user.ID,
		TenantID: stringPtrValue(user.TenantID),
		Role:     string(user.Role),
		Key:      s.cfg.KeyID,
		RegisteredClaims: jwtRegisteredClaims(
			user.ID, s.cfg.Issuer, uuid.New().String(), now, accessExpiresAt,
		),
	}

	accessToken, err := s.signer.Sign(accessClaims)
	if err != nil {
		return nil, err
	}

	refreshJTI := uuid.New().String()
	refreshClaims := domain.Claims{
		TokenUse: domain.TokenUseRefresh,
		RegisteredClaims: jwtRegisteredClaims(
			user.ID, s.cfg.Issuer, refreshJTI, now, refreshExpiresAt,
		),
	}

	refreshToken, err := s.signer.Sign(refreshClaims)
	if err != nil {
		return nil, err
	}

	if err := s.refreshRepo.Save(context.WithoutCancel(ctx), refreshJTI, user.ID, s.refreshTTL); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func jwtRegisteredClaims(sub, iss, jti string, issuedAt, expiresAt time.Time) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Subject:   sub,
		Issuer:    iss,
		ID:        jti,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
}

func stringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func isSmsCredentialError(err error) bool {
	return errors.Is(err, smserrs.ErrCodeNotFound) ||
		errors.Is(err, smserrs.ErrCodeExpired) ||
		errors.Is(err, smserrs.ErrCodeUsed) ||
		errors.Is(err, smserrs.ErrCodeMismatch)
}
