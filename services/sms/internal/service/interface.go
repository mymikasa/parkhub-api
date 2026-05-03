package service

import (
	"context"

	"github.com/parkhub/api/services/sms/internal/domain"
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

type SmsService interface {
	SendCode(ctx context.Context, req *SendCodeRequest) error
	VerifyCode(ctx context.Context, req *VerifyCodeRequest) error
}
