package repository

import (
	"context"
	"time"

	"github.com/parkhub/api/services/sms/internal/domain"
)

type SmsRepository interface {
	SaveCode(ctx context.Context, code *domain.SmsCode) error
	SaveSendFailure(ctx context.Context, phone string, purpose domain.SmsPurpose, providerErr string) error
	GetCode(ctx context.Context, phone string, purpose domain.SmsPurpose) (*domain.SmsCode, error)
	VerifyAndConsume(ctx context.Context, phone string, purpose domain.SmsPurpose, input string) error
	TryReserveRateLimit(ctx context.Context, phone string, ttl time.Duration) (bool, error)
	ReleaseRateLimit(ctx context.Context, phone string) error
}
