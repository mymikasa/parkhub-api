package grpc

import (
	"context"
	"net"
	"testing"

	"github.com/parkhub/api/internal/domains/sms/domain"
	"github.com/parkhub/api/internal/domains/sms/errs"
	servicemocks "github.com/parkhub/api/internal/domains/sms/service/mocks"
	smsv1 "github.com/parkhub/api/internal/gen/api/proto/sms/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	go_mock "go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func setupSmsTestServer(t *testing.T) (smsv1.SmsServiceClient, *servicemocks.MockSmsService, *go_mock.Controller) {
	t.Helper()
	ctrl := go_mock.NewController(t)
	mockSvc := servicemocks.NewMockSmsService(ctrl)

	srv := NewSmsGRPCServer(mockSvc)
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	smsv1.RegisterSmsServiceServer(s, srv)
	go s.Serve(lis)
	t.Cleanup(s.GracefulStop)

	conn, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return smsv1.NewSmsServiceClient(conn), mockSvc, ctrl
}

func TestGRPC_SendCode_Success(t *testing.T) {
	client, mockSvc, ctrl := setupSmsTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().SendCode(go_mock.Any(), go_mock.Any()).Return(nil)

	resp, err := client.SendCode(context.Background(), &smsv1.SendCodeRequest{
		Phone:   "13800138000",
		Purpose: smsv1.SmsPurpose_SMS_PURPOSE_LOGIN,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestGRPC_SendCode_RateLimited(t *testing.T) {
	client, mockSvc, ctrl := setupSmsTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().SendCode(go_mock.Any(), go_mock.Any()).Return(errs.ErrPhoneRateLimit)

	_, err := client.SendCode(context.Background(), &smsv1.SendCodeRequest{
		Phone:   "13800138000",
		Purpose: smsv1.SmsPurpose_SMS_PURPOSE_LOGIN,
	})
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestGRPC_SendCode_InvalidPhone(t *testing.T) {
	client, mockSvc, ctrl := setupSmsTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().SendCode(go_mock.Any(), go_mock.Any()).Return(errs.ErrInvalidPhone)

	_, err := client.SendCode(context.Background(), &smsv1.SendCodeRequest{
		Phone:   "123",
		Purpose: smsv1.SmsPurpose_SMS_PURPOSE_LOGIN,
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPC_VerifyCode_Success(t *testing.T) {
	client, mockSvc, ctrl := setupSmsTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().VerifyCode(go_mock.Any(), go_mock.Any()).Return(nil)

	resp, err := client.VerifyCode(context.Background(), &smsv1.VerifyCodeRequest{
		Phone:   "13800138000",
		Code:    "123456",
		Purpose: smsv1.SmsPurpose_SMS_PURPOSE_LOGIN,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestGRPC_VerifyCode_Mismatch(t *testing.T) {
	client, mockSvc, ctrl := setupSmsTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().VerifyCode(go_mock.Any(), go_mock.Any()).Return(errs.ErrCodeMismatch)

	_, err := client.VerifyCode(context.Background(), &smsv1.VerifyCodeRequest{
		Phone:   "13800138000",
		Code:    "000000",
		Purpose: smsv1.SmsPurpose_SMS_PURPOSE_LOGIN,
	})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestGRPC_VerifyCode_Expired(t *testing.T) {
	client, mockSvc, ctrl := setupSmsTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().VerifyCode(go_mock.Any(), go_mock.Any()).Return(errs.ErrCodeExpired)

	_, err := client.VerifyCode(context.Background(), &smsv1.VerifyCodeRequest{
		Phone:   "13800138000",
		Code:    "123456",
		Purpose: smsv1.SmsPurpose_SMS_PURPOSE_LOGIN,
	})
	assert.Equal(t, codes.DeadlineExceeded, status.Code(err))
}

func TestGRPC_VerifyCode_Used(t *testing.T) {
	client, mockSvc, ctrl := setupSmsTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().VerifyCode(go_mock.Any(), go_mock.Any()).Return(errs.ErrCodeUsed)

	_, err := client.VerifyCode(context.Background(), &smsv1.VerifyCodeRequest{
		Phone:   "13800138000",
		Code:    "123456",
		Purpose: smsv1.SmsPurpose_SMS_PURPOSE_LOGIN,
	})
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGRPC_VerifyCode_NotFound(t *testing.T) {
	client, mockSvc, ctrl := setupSmsTestServer(t)
	defer ctrl.Finish()

	mockSvc.EXPECT().VerifyCode(go_mock.Any(), go_mock.Any()).Return(errs.ErrCodeNotFound)

	_, err := client.VerifyCode(context.Background(), &smsv1.VerifyCodeRequest{
		Phone:   "13800138000",
		Code:    "123456",
		Purpose: smsv1.SmsPurpose_SMS_PURPOSE_LOGIN,
	})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestPurposeToProto_AllCases(t *testing.T) {
	tests := []struct {
		input    domain.SmsPurpose
		expected smsv1.SmsPurpose
	}{
		{domain.SmsPurposeLogin, smsv1.SmsPurpose_SMS_PURPOSE_LOGIN},
		{domain.SmsPurposeRegister, smsv1.SmsPurpose_SMS_PURPOSE_REGISTER},
		{domain.SmsPurposeResetPassword, smsv1.SmsPurpose_SMS_PURPOSE_RESET_PASSWORD},
		{"unknown", smsv1.SmsPurpose_SMS_PURPOSE_UNSPECIFIED},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, purposeToProto(tt.input), "input: %s", tt.input)
	}
}

func TestPurposeFromProto_AllCases(t *testing.T) {
	tests := []struct {
		input    smsv1.SmsPurpose
		expected domain.SmsPurpose
	}{
		{smsv1.SmsPurpose_SMS_PURPOSE_LOGIN, domain.SmsPurposeLogin},
		{smsv1.SmsPurpose_SMS_PURPOSE_REGISTER, domain.SmsPurposeRegister},
		{smsv1.SmsPurpose_SMS_PURPOSE_RESET_PASSWORD, domain.SmsPurposeResetPassword},
		{smsv1.SmsPurpose_SMS_PURPOSE_UNSPECIFIED, ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, purposeFromProto(tt.input), "input: %v", tt.input)
	}
}
