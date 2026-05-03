package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/parkhub/api/services/sms/internal/domain"
	"github.com/parkhub/api/services/sms/internal/repository/cache"
	"github.com/parkhub/api/services/sms/internal/repository/dao"
)

type smsRepository struct {
	dao   dao.SmsRecordDAO
	cache cache.SmsCache
}

func NewSmsRepository(dao dao.SmsRecordDAO, cache cache.SmsCache) SmsRepository {
	return &smsRepository{dao: dao, cache: cache}
}

func (r *smsRepository) SaveCode(ctx context.Context, code *domain.SmsCode) error {
	if err := r.cache.Store(ctx, code); err != nil {
		return err
	}

	record := &dao.SmsRecord{
		ID:      uuid.New().String(),
		Phone:   code.Phone,
		Purpose: string(code.Purpose),
		Code:    code.Code,
		Status:  "success",
	}
	return r.dao.Insert(ctx, record)
}

func (r *smsRepository) SaveSendFailure(ctx context.Context, phone string, purpose domain.SmsPurpose, providerErr string) error {
	record := &dao.SmsRecord{
		ID:          uuid.New().String(),
		Phone:       phone,
		Purpose:     string(purpose),
		Code:        "",
		Status:      "failed",
		ProviderErr: &providerErr,
	}
	return r.dao.Insert(ctx, record)
}

func (r *smsRepository) GetCode(ctx context.Context, phone string, purpose domain.SmsPurpose) (*domain.SmsCode, error) {
	return r.cache.Retrieve(ctx, phone, purpose)
}

func (r *smsRepository) MarkCodeUsed(ctx context.Context, phone string, purpose domain.SmsPurpose) error {
	return r.cache.VerifyAndConsume(ctx, phone, purpose, "")
}

func (r *smsRepository) VerifyAndConsume(ctx context.Context, phone string, purpose domain.SmsPurpose, input string) error {
	return r.cache.VerifyAndConsume(ctx, phone, purpose, input)
}

func (r *smsRepository) TryReserveRateLimit(ctx context.Context, phone string, ttl time.Duration) (bool, error) {
	return r.cache.TryReserveRateLimit(ctx, phone, ttl)
}

func (r *smsRepository) ReleaseRateLimit(ctx context.Context, phone string) error {
	return r.cache.ReleaseRateLimit(ctx, phone)
}
