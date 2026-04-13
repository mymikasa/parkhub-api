package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/parkhub/api/internal/domains/sms/domain"
	"github.com/parkhub/api/internal/domains/sms/errs"
	"github.com/parkhub/api/internal/domains/sms/gateway"
	repomocks "github.com/parkhub/api/internal/domains/sms/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	go_mock "go.uber.org/mock/gomock"
)

func setupSmsService(t *testing.T) (SmsService, *repomocks.MockSmsRepository, *gateway.MockSmsGateway) {
	t.Helper()
	ctrl := go_mock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockRepo := repomocks.NewMockSmsRepository(ctrl)
	mockGW := gateway.NewMockSmsGateway()
	return NewSmsService(mockRepo, mockGW), mockRepo, mockGW
}

func setupSmsServiceWithMockGW(t *testing.T) (SmsService, *repomocks.MockSmsRepository, *gateway.MockSmsGateway) {
	t.Helper()
	ctrl := go_mock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockRepo := repomocks.NewMockSmsRepository(ctrl)
	mockGW := gateway.NewMockSmsGateway()
	return NewSmsService(mockRepo, mockGW), mockRepo, mockGW
}

func TestSmsService_SendCode_Success(t *testing.T) {
	svc, mockRepo, mockGW := setupSmsServiceWithMockGW(t)
	ctx := context.Background()

	mockRepo.EXPECT().CheckRateLimit(go_mock.Any(), "13800138000").Return(false, nil)
	mockRepo.EXPECT().SaveCode(go_mock.Any(), go_mock.Any()).Return(nil)
	mockRepo.EXPECT().SetRateLimit(go_mock.Any(), "13800138000", 60*time.Second).Return(nil)

	err := svc.SendCode(ctx, &SendCodeRequest{
		Phone:   "13800138000",
		Purpose: domain.SmsPurposeLogin,
	})
	require.NoError(t, err)
	require.Len(t, mockGW.Calls, 1)
	assert.Equal(t, "13800138000", mockGW.Calls[0].Phone)
}

func TestSmsService_SendCode_InvalidPhone(t *testing.T) {
	svc, _, _ := setupSmsService(t)
	ctx := context.Background()

	err := svc.SendCode(ctx, &SendCodeRequest{
		Phone:   "123",
		Purpose: domain.SmsPurposeLogin,
	})
	assert.ErrorIs(t, err, errs.ErrInvalidPhone)
}

func TestSmsService_SendCode_RateLimited(t *testing.T) {
	svc, mockRepo, _ := setupSmsService(t)
	ctx := context.Background()

	mockRepo.EXPECT().CheckRateLimit(go_mock.Any(), "13800138000").Return(true, nil)

	err := svc.SendCode(ctx, &SendCodeRequest{
		Phone:   "13800138000",
		Purpose: domain.SmsPurposeLogin,
	})
	assert.ErrorIs(t, err, errs.ErrPhoneRateLimit)
}

func TestSmsService_SendCode_GatewayFailure(t *testing.T) {
	ctrl := go_mock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockRepo := repomocks.NewMockSmsRepository(ctrl)
	mockGW := &gateway.MockSmsGateway{Err: errors.New("gateway timeout")}

	svc := NewSmsService(mockRepo, mockGW)
	ctx := context.Background()

	mockRepo.EXPECT().CheckRateLimit(go_mock.Any(), "13800138000").Return(false, nil)
	mockRepo.EXPECT().SaveSendFailure(go_mock.Any(), "13800138000", domain.SmsPurposeLogin, "gateway timeout").Return(nil)

	err := svc.SendCode(ctx, &SendCodeRequest{
		Phone:   "13800138000",
		Purpose: domain.SmsPurposeLogin,
	})
	require.NoError(t, err)
}

func TestSmsService_VerifyCode_Success(t *testing.T) {
	svc, mockRepo, _ := setupSmsService(t)
	ctx := context.Background()

	mockRepo.EXPECT().VerifyAndConsume(go_mock.Any(), "13800138000", domain.SmsPurposeLogin, "123456").Return(nil)

	err := svc.VerifyCode(ctx, &VerifyCodeRequest{
		Phone:   "13800138000",
		Code:    "123456",
		Purpose: domain.SmsPurposeLogin,
	})
	assert.NoError(t, err)
}

func TestSmsService_VerifyCode_Mismatch(t *testing.T) {
	svc, mockRepo, _ := setupSmsService(t)
	ctx := context.Background()

	mockRepo.EXPECT().VerifyAndConsume(go_mock.Any(), "13800138000", domain.SmsPurposeLogin, "000000").Return(errs.ErrCodeMismatch)

	err := svc.VerifyCode(ctx, &VerifyCodeRequest{
		Phone:   "13800138000",
		Code:    "000000",
		Purpose: domain.SmsPurposeLogin,
	})
	assert.ErrorIs(t, err, errs.ErrCodeMismatch)
}

func TestSmsService_VerifyCode_NotFound(t *testing.T) {
	svc, mockRepo, _ := setupSmsService(t)
	ctx := context.Background()

	mockRepo.EXPECT().VerifyAndConsume(go_mock.Any(), "13800138000", domain.SmsPurposeLogin, "123456").Return(errs.ErrCodeNotFound)

	err := svc.VerifyCode(ctx, &VerifyCodeRequest{
		Phone:   "13800138000",
		Code:    "123456",
		Purpose: domain.SmsPurposeLogin,
	})
	assert.ErrorIs(t, err, errs.ErrCodeNotFound)
}
