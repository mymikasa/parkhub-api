package service

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/parkhub/api/internal/domains/sms/domain"
	"github.com/parkhub/api/internal/domains/sms/errs"
	"github.com/parkhub/api/internal/domains/sms/gateway"
	"github.com/parkhub/api/internal/domains/sms/repository"
)

var phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)

const (
	codeTTL      = 5 * time.Minute
	rateLimitTTL = 60 * time.Second
)

type smsService struct {
	repo    repository.SmsRepository
	gateway gateway.SmsGateway
}

func NewSmsService(repo repository.SmsRepository, gw gateway.SmsGateway) SmsService {
	return &smsService{repo: repo, gateway: gw}
}

func (s *smsService) SendCode(ctx context.Context, req *SendCodeRequest) error {
	if !isValidPhone(req.Phone) {
		return errs.ErrInvalidPhone
	}

	reserved, err := s.repo.TryReserveRateLimit(ctx, req.Phone, rateLimitTTL)
	if err != nil {
		return err
	}
	if !reserved {
		return errs.ErrPhoneRateLimit
	}

	code, err := domain.NewSmsCode(req.Phone, req.Purpose, codeTTL)
	if err != nil {
		_ = s.repo.ReleaseRateLimit(ctx, req.Phone)
		return err
	}
	code.ID = uuid.New().String()

	if err := s.gateway.Send(ctx, code.Phone, code.Code, string(code.Purpose)); err != nil {
		_ = s.repo.ReleaseRateLimit(ctx, req.Phone)
		return s.repo.SaveSendFailure(ctx, req.Phone, req.Purpose, err.Error())
	}

	return s.repo.SaveCode(ctx, code)
}

func (s *smsService) VerifyCode(ctx context.Context, req *VerifyCodeRequest) error {
	return s.repo.VerifyAndConsume(ctx, req.Phone, req.Purpose, req.Code)
}

func isValidPhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}
