package domain

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenUseAccess  = "access"
	TokenUseRefresh = "refresh"
)

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type Claims struct {
	TokenUse string `json:"token_use"`
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id,omitempty"`
	Role     string `json:"role,omitempty"`
	Key      string `json:"key,omitempty"`
	jwt.RegisteredClaims
}
