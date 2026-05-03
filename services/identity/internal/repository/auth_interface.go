package repository

import (
	"context"
	"time"
)

type RefreshTokenRepo interface {
	Save(ctx context.Context, jti, userID string, ttl time.Duration) error
	Consume(ctx context.Context, jti string) (userID string, ok bool, err error)
	Revoke(ctx context.Context, jti string) error
}
