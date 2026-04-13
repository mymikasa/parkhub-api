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

	limited, err := s.repo.CheckRateLimit(ctx, req.Phone)
	if err != nil {
		return err
	}
	if limited {
		return errs.ErrPhoneRateLimit
	}

	code, err := domain.NewSmsCode(req.Phone, req.Purpose, codeTTL)
	if err != nil {
		return err
	}
	code.ID = uuid.New().String()

	if err := s.gateway.Send(ctx, code.Phone, code.Code, string(code.Purpose)); err != nil {
		return s.repo.SaveSendFailure(ctx, req.Phone, req.Purpose, err.Error())
	}

	if err := s.repo.SaveCode(ctx, code); err != nil {
		return err
	}

	return s.repo.SetRateLimit(ctx, req.Phone, rateLimitTTL)
}

func (s *smsService) VerifyCode(ctx context.Context, req *VerifyCodeRequest) error {
	code, err := s.repo.GetCode(ctx, req.Phone, req.Purpose)
	if err != nil {
		return err
	}

	if err := code.Verify(req.Code); err != nil {
		return err
	}

	return s.repo.MarkCodeUsed(ctx, req.Phone, req.Purpose)
}

func isValidPhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}
