package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/parkhub/api/internal/domains/sms/domain"
	"github.com/parkhub/api/internal/domains/sms/repository/cache"
	"github.com/parkhub/api/internal/domains/sms/repository/dao"
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
	return r.cache.MarkUsed(ctx, phone, purpose)
}

func (r *smsRepository) SetRateLimit(ctx context.Context, phone string, ttl time.Duration) error {
	return r.cache.SetRateLimit(ctx, phone, ttl)
}

func (r *smsRepository) CheckRateLimit(ctx context.Context, phone string) (bool, error) {
	return r.cache.CheckRateLimit(ctx, phone)
}
