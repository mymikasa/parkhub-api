package service

import (
	"context"

	"github.com/parkhub/api/internal/domains/sms/domain"
)

type SendCodeRequest struct {
	Phone   string
	Purpose domain.SmsPurpose
}

type VerifyCodeRequest struct {
	Phone   string
	Code    string
	Purpose domain.SmsPurpose
}

//go:generate mockgen -source=./interface.go -package=servicemocks -destination=./mocks/sms_service.mock.go SmsService

type SmsService interface {
	SendCode(ctx context.Context, req *SendCodeRequest) error
	VerifyCode(ctx context.Context, req *VerifyCodeRequest) error
}
