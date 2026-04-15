package service

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/parkhub/api/internal/config"
	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	repomocks "github.com/parkhub/api/internal/domains/identity/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func loadTestSigner(t *testing.T) *domain.RS256Signer {
	t.Helper()
	privPEM, err := os.ReadFile("../../../../configs/keys/jwt_private.pem")
	require.NoError(t, err)
	pubPEM, err := os.ReadFile("../../../../configs/keys/jwt_public.pem")
	require.NoError(t, err)
	signer, err := domain.NewRS256Signer(privPEM, pubPEM, "test-key", "parkhub")
	require.NoError(t, err)
	return signer
}

func testAuthCfg() config.AuthConfig {
	return config.AuthConfig{
		Issuer:     "parkhub",
		AccessTTL:  "15m",
		RefreshTTL: "168h",
		KeyID:      "test-key",
	}
}

func setupAuthService(t *testing.T) (AuthService, *repomocks.MockUserRepo, *repomocks.MockRefreshTokenRepo) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockUserRepo := repomocks.NewMockUserRepo(ctrl)
	mockRefreshRepo := repomocks.NewMockRefreshTokenRepo(ctrl)
	signer := loadTestSigner(t)
	cfg := testAuthCfg()
	svc := NewAuthService(mockUserRepo, mockRefreshRepo, signer, cfg)
	return svc, mockUserRepo, mockRefreshRepo
}

func hashedPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	return string(hash)
}

func newTestUser() *domain.User {
	tenantID := "tenant-1"
	return &domain.User{
		ID:           "user-1",
		TenantID:     &tenantID,
		Username:     "admin",
		PasswordHash: "",
		RealName:     "Admin",
		Role:         domain.RoleTenantAdmin,
		Status:       domain.UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	svc, mockUserRepo, mockRefreshRepo := setupAuthService(t)
	ctx := context.Background()

	user := newTestUser()
	user.PasswordHash = hashedPassword(t, "password123")

	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), "admin").Return(user, nil)
	mockRefreshRepo.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockUserRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	resp, err := svc.Login(ctx, &LoginRequest{Username: "admin", Password: "password123"})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.TokenPair.AccessToken)
	assert.NotEmpty(t, resp.TokenPair.RefreshToken)
	assert.Equal(t, "user-1", resp.User.ID)
	assert.Equal(t, int32(900), resp.AccessExpiresIn)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc, mockUserRepo, _ := setupAuthService(t)
	ctx := context.Background()

	user := newTestUser()
	user.PasswordHash = hashedPassword(t, "password123")

	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), "admin").Return(user, nil)

	_, err := svc.Login(ctx, &LoginRequest{Username: "admin", Password: "wrong"})
	assert.ErrorIs(t, err, errs.ErrInvalidCredentials)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	svc, mockUserRepo, _ := setupAuthService(t)
	ctx := context.Background()

	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), "admin").Return(nil, errs.ErrUserNotFound)

	_, err := svc.Login(ctx, &LoginRequest{Username: "admin", Password: "password123"})
	assert.ErrorIs(t, err, errs.ErrInvalidCredentials)
}

func TestAuthService_Login_FrozenUser(t *testing.T) {
	svc, mockUserRepo, _ := setupAuthService(t)
	ctx := context.Background()

	user := newTestUser()
	user.PasswordHash = hashedPassword(t, "password123")
	user.Status = domain.UserStatusFrozen

	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), "admin").Return(user, nil)

	_, err := svc.Login(ctx, &LoginRequest{Username: "admin", Password: "password123"})
	assert.ErrorIs(t, err, errs.ErrUserFrozen)
}

func TestAuthService_Login_AccessTokenContainsClaims(t *testing.T) {
	svc, mockUserRepo, mockRefreshRepo := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	user := newTestUser()
	user.PasswordHash = hashedPassword(t, "password123")

	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), "admin").Return(user, nil)
	mockRefreshRepo.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockUserRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	resp, err := svc.Login(ctx, &LoginRequest{Username: "admin", Password: "password123"})
	require.NoError(t, err)

	claims, err := signer.Verify(resp.TokenPair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, domain.TokenUseAccess, claims.TokenUse)
	assert.Equal(t, "user-1", claims.UserID)
	assert.Equal(t, "tenant-1", claims.TenantID)
	assert.Equal(t, "tenant_admin", claims.Role)
}

func TestAuthService_Login_RefreshTokenHasMinimalClaims(t *testing.T) {
	svc, mockUserRepo, mockRefreshRepo := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	user := newTestUser()
	user.PasswordHash = hashedPassword(t, "password123")

	mockUserRepo.EXPECT().GetByUsername(gomock.Any(), "admin").Return(user, nil)
	mockRefreshRepo.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockUserRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	resp, err := svc.Login(ctx, &LoginRequest{Username: "admin", Password: "password123"})
	require.NoError(t, err)

	claims, err := signer.Verify(resp.TokenPair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, domain.TokenUseRefresh, claims.TokenUse)
	assert.NotEmpty(t, claims.ID)
}

func TestAuthService_RefreshToken_Success(t *testing.T) {
	svc, mockUserRepo, mockRefreshRepo := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	jti := uuid.New().String()
	refreshClaims := domain.Claims{
		TokenUse: domain.TokenUseRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    "parkhub",
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(168 * time.Hour)),
		},
	}
	refreshToken, err := signer.Sign(refreshClaims)
	require.NoError(t, err)

	user := newTestUser()
	mockRefreshRepo.EXPECT().Consume(gomock.Any(), jti).Return("user-1", true, nil)
	mockUserRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(user, nil)
	mockRefreshRepo.EXPECT().Save(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	resp, err := svc.RefreshToken(ctx, &RefreshTokenRequest{RefreshToken: refreshToken})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.TokenPair.AccessToken)
	assert.NotEmpty(t, resp.TokenPair.RefreshToken)
}

func TestAuthService_RefreshToken_ExpiredToken(t *testing.T) {
	svc, _, _ := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	jti := uuid.New().String()
	refreshClaims := domain.Claims{
		TokenUse: domain.TokenUseRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    "parkhub",
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	refreshToken, err := signer.Sign(refreshClaims)
	require.NoError(t, err)

	_, err = svc.RefreshToken(ctx, &RefreshTokenRequest{RefreshToken: refreshToken})
	assert.ErrorIs(t, err, errs.ErrRefreshTokenInvalid)
}

func TestAuthService_RefreshToken_RevokedToken(t *testing.T) {
	svc, _, mockRefreshRepo := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	jti := uuid.New().String()
	refreshClaims := domain.Claims{
		TokenUse: domain.TokenUseRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    "parkhub",
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(168 * time.Hour)),
		},
	}
	refreshToken, err := signer.Sign(refreshClaims)
	require.NoError(t, err)

	mockRefreshRepo.EXPECT().Consume(gomock.Any(), jti).Return("", false, nil)

	_, err = svc.RefreshToken(ctx, &RefreshTokenRequest{RefreshToken: refreshToken})
	assert.ErrorIs(t, err, errs.ErrRefreshTokenRevoked)
}

func TestAuthService_RefreshToken_Replay(t *testing.T) {
	svc, _, mockRefreshRepo := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	jti := uuid.New().String()
	refreshClaims := domain.Claims{
		TokenUse: domain.TokenUseRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    "parkhub",
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(168 * time.Hour)),
		},
	}
	refreshToken, err := signer.Sign(refreshClaims)
	require.NoError(t, err)

	mockRefreshRepo.EXPECT().Consume(gomock.Any(), jti).Return("", false, nil)

	_, err = svc.RefreshToken(ctx, &RefreshTokenRequest{RefreshToken: refreshToken})
	assert.ErrorIs(t, err, errs.ErrRefreshTokenRevoked)
}

func TestAuthService_RefreshToken_AccessTokenMisused(t *testing.T) {
	svc, _, _ := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	accessClaims := domain.Claims{
		TokenUse: domain.TokenUseAccess,
		UserID:   "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    "parkhub",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	accessToken, err := signer.Sign(accessClaims)
	require.NoError(t, err)

	_, err = svc.RefreshToken(ctx, &RefreshTokenRequest{RefreshToken: accessToken})
	assert.ErrorIs(t, err, errs.ErrRefreshTokenInvalid)
}

func TestAuthService_RefreshToken_FrozenUser(t *testing.T) {
	svc, mockUserRepo, mockRefreshRepo := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	jti := uuid.New().String()
	refreshClaims := domain.Claims{
		TokenUse: domain.TokenUseRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    "parkhub",
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(168 * time.Hour)),
		},
	}
	refreshToken, err := signer.Sign(refreshClaims)
	require.NoError(t, err)

	user := newTestUser()
	user.Status = domain.UserStatusFrozen

	mockRefreshRepo.EXPECT().Consume(gomock.Any(), jti).Return("user-1", true, nil)
	mockUserRepo.EXPECT().GetByID(gomock.Any(), "user-1").Return(user, nil)

	_, err = svc.RefreshToken(ctx, &RefreshTokenRequest{RefreshToken: refreshToken})
	assert.ErrorIs(t, err, errs.ErrUserFrozen)
}

func TestAuthService_Logout_Success(t *testing.T) {
	svc, _, mockRefreshRepo := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	jti := uuid.New().String()
	refreshClaims := domain.Claims{
		TokenUse: domain.TokenUseRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    "parkhub",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(168 * time.Hour)),
		},
	}
	refreshToken, err := signer.Sign(refreshClaims)
	require.NoError(t, err)

	mockRefreshRepo.EXPECT().Revoke(gomock.Any(), jti).Return(nil)

	err = svc.Logout(ctx, &LogoutRequest{RefreshToken: refreshToken})
	assert.NoError(t, err)
}

func TestAuthService_Logout_Idempotent(t *testing.T) {
	svc, _, mockRefreshRepo := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	jti := uuid.New().String()
	refreshClaims := domain.Claims{
		TokenUse: domain.TokenUseRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    "parkhub",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(168 * time.Hour)),
		},
	}
	refreshToken, err := signer.Sign(refreshClaims)
	require.NoError(t, err)

	mockRefreshRepo.EXPECT().Revoke(gomock.Any(), jti).Return(nil)

	err = svc.Logout(ctx, &LogoutRequest{RefreshToken: refreshToken})
	assert.NoError(t, err)
}

func TestAuthService_Logout_InvalidToken(t *testing.T) {
	svc, _, _ := setupAuthService(t)
	ctx := context.Background()

	err := svc.Logout(ctx, &LogoutRequest{RefreshToken: "invalid-token"})
	assert.ErrorIs(t, err, errs.ErrRefreshTokenInvalid)
}

func TestAuthService_Logout_AccessTokenMisused(t *testing.T) {
	svc, _, _ := setupAuthService(t)
	ctx := context.Background()
	signer := loadTestSigner(t)

	accessClaims := domain.Claims{
		TokenUse: domain.TokenUseAccess,
		UserID:   "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    "parkhub",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	accessToken, err := signer.Sign(accessClaims)
	require.NoError(t, err)

	err = svc.Logout(ctx, &LogoutRequest{RefreshToken: accessToken})
	assert.ErrorIs(t, err, errs.ErrRefreshTokenInvalid)
}

func TestParseTTL(t *testing.T) {
	assert.Equal(t, 15*time.Minute, ParseTTL("15m"))
	assert.Equal(t, 168*time.Hour, ParseTTL("168h"))
	assert.Equal(t, 15*time.Minute, ParseTTL("invalid"))
}

func TestRS256Signer_JWKS(t *testing.T) {
	signer := loadTestSigner(t)

	jwks, err := signer.JWKS()
	require.NoError(t, err)
	assert.Contains(t, string(jwks), `"kid":"test-key"`)
	assert.Contains(t, string(jwks), `"kty":"RSA"`)
	assert.Contains(t, string(jwks), `"alg":"RS256"`)
}

func TestRS256Signer_Verify_InvalidSignature(t *testing.T) {
	signer := loadTestSigner(t)

	privPEM, _ := os.ReadFile("../../../../configs/keys/jwt_private.pem")
	block, _ := pem.Decode(privPEM)
	privKey, _ := x509.ParsePKCS8PrivateKey(block.Bytes)
	rsaKey := privKey.(*rsa.PrivateKey)

	claims := domain.Claims{
		TokenUse: domain.TokenUseAccess,
		UserID:   "user-1",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    "parkhub",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key"
	signed, _ := token.SignedString(rsaKey)

	_, err := signer.Verify(signed + "tampered")
	assert.Error(t, err)
}
