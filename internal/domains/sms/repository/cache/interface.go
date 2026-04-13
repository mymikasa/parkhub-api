package cache

import (
	"context"
	"time"

	"github.com/parkhub/api/internal/domains/sms/domain"
)

type SmsCache interface {
	Store(ctx context.Context, code *domain.SmsCode) error
	Retrieve(ctx context.Context, phone string, purpose domain.SmsPurpose) (*domain.SmsCode, error)
	MarkUsed(ctx context.Context, phone string, purpose domain.SmsPurpose) error
	SetRateLimit(ctx context.Context, phone string, ttl time.Duration) error
	CheckRateLimit(ctx context.Context, phone string) (bool, error)
}
