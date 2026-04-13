package repository

import (
	"context"
	"time"

	"github.com/parkhub/api/internal/domains/sms/domain"
)

//go:generate mockgen -source=./interface.go -package=repomocks -destination=./mocks/repo.mock.go SmsRepository

type SmsRepository interface {
	SaveCode(ctx context.Context, code *domain.SmsCode) error
	SaveSendFailure(ctx context.Context, phone string, purpose domain.SmsPurpose, providerErr string) error
	GetCode(ctx context.Context, phone string, purpose domain.SmsPurpose) (*domain.SmsCode, error)
	VerifyAndConsume(ctx context.Context, phone string, purpose domain.SmsPurpose, input string) error
	SetRateLimit(ctx context.Context, phone string, ttl time.Duration) error
	CheckRateLimit(ctx context.Context, phone string) (bool, error)
}
