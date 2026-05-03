package cache

import (
	"context"
	"time"

	"github.com/parkhub/api/services/sms/internal/domain"
)

type SmsCache interface {
	Store(ctx context.Context, code *domain.SmsCode) error
	Retrieve(ctx context.Context, phone string, purpose domain.SmsPurpose) (*domain.SmsCode, error)
	VerifyAndConsume(ctx context.Context, phone string, purpose domain.SmsPurpose, input string) error
	TryReserveRateLimit(ctx context.Context, phone string, ttl time.Duration) (bool, error)
	ReleaseRateLimit(ctx context.Context, phone string) error
}
