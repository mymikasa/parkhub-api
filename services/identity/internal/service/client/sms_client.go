package client

import (
	"context"

	smsv1 "github.com/parkhub/api/services/identity/internal/gen/api/proto/sms/v1"
	"github.com/parkhub/api/services/identity/internal/service"
)

var _ service.SmsCodeVerifier = (*SmsVerifierClient)(nil)

type SmsVerifierClient struct {
	client smsv1.SmsServiceClient
}

func NewSmsVerifierClient(client smsv1.SmsServiceClient) *SmsVerifierClient {
	return &SmsVerifierClient{client: client}
}

func (c *SmsVerifierClient) VerifyCode(ctx context.Context, phone, code string) error {
	_, err := c.client.VerifyCode(ctx, &smsv1.VerifyCodeRequest{
		Phone:   phone,
		Code:    code,
		Purpose: smsv1.SmsPurpose_SMS_PURPOSE_LOGIN,
	})
	return err
}
