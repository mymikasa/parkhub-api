package grpc

import (
	"context"

	"github.com/parkhub/api/pkg/grpcutil"
	"github.com/parkhub/api/services/sms/internal/errs"
	smsv1 "github.com/parkhub/api/services/sms/internal/gen/api/proto/sms/v1"
	"github.com/parkhub/api/services/sms/internal/service"
	"google.golang.org/grpc/codes"
)

type SmsGRPCServer struct {
	smsv1.UnimplementedSmsServiceServer
	smsSvc service.SmsService
}

func NewSmsGRPCServer(svc service.SmsService) *SmsGRPCServer {
	return &SmsGRPCServer{smsSvc: svc}
}

var smsErrorMappings = []grpcutil.ErrorMapping{
	{Target: errs.ErrCodeNotFound, Code: codes.NotFound},
	{Target: errs.ErrCodeExpired, Code: codes.DeadlineExceeded},
	{Target: errs.ErrCodeUsed, Code: codes.FailedPrecondition},
	{Target: errs.ErrCodeMismatch, Code: codes.Unauthenticated},
	{Target: errs.ErrPhoneRateLimit, Code: codes.ResourceExhausted},
	{Target: errs.ErrInvalidPhone, Code: codes.InvalidArgument},
	{Target: errs.ErrInvalidPurpose, Code: codes.InvalidArgument},
}

func toSmsGRPCError(err error) error {
	return grpcutil.ToGRPCError(err, smsErrorMappings)
}

func (s *SmsGRPCServer) SendCode(ctx context.Context, req *smsv1.SendCodeRequest) (*smsv1.SendCodeResponse, error) {
	purpose := purposeFromProto(req.Purpose)
	if purpose == "" {
		return nil, toSmsGRPCError(errs.ErrInvalidPurpose)
	}

	err := s.smsSvc.SendCode(ctx, &service.SendCodeRequest{
		Phone:   req.Phone,
		Purpose: purpose,
	})
	if err != nil {
		return nil, toSmsGRPCError(err)
	}
	return &smsv1.SendCodeResponse{}, nil
}

func (s *SmsGRPCServer) VerifyCode(ctx context.Context, req *smsv1.VerifyCodeRequest) (*smsv1.VerifyCodeResponse, error) {
	purpose := purposeFromProto(req.Purpose)
	if purpose == "" {
		return nil, toSmsGRPCError(errs.ErrInvalidPurpose)
	}

	err := s.smsSvc.VerifyCode(ctx, &service.VerifyCodeRequest{
		Phone:   req.Phone,
		Code:    req.Code,
		Purpose: purpose,
	})
	if err != nil {
		return nil, toSmsGRPCError(err)
	}
	return &smsv1.VerifyCodeResponse{}, nil
}
