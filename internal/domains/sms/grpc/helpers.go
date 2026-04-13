package grpc

import (
	"github.com/parkhub/api/internal/domains/sms/domain"
	smsv1 "github.com/parkhub/api/internal/gen/api/proto/sms/v1"
)

func purposeFromProto(p smsv1.SmsPurpose) domain.SmsPurpose {
	switch p {
	case smsv1.SmsPurpose_SMS_PURPOSE_LOGIN:
		return domain.SmsPurposeLogin
	case smsv1.SmsPurpose_SMS_PURPOSE_REGISTER:
		return domain.SmsPurposeRegister
	case smsv1.SmsPurpose_SMS_PURPOSE_RESET_PASSWORD:
		return domain.SmsPurposeResetPassword
	default:
		return ""
	}
}

func purposeToProto(p domain.SmsPurpose) smsv1.SmsPurpose {
	switch p {
	case domain.SmsPurposeLogin:
		return smsv1.SmsPurpose_SMS_PURPOSE_LOGIN
	case domain.SmsPurposeRegister:
		return smsv1.SmsPurpose_SMS_PURPOSE_REGISTER
	case domain.SmsPurposeResetPassword:
		return smsv1.SmsPurpose_SMS_PURPOSE_RESET_PASSWORD
	default:
		return smsv1.SmsPurpose_SMS_PURPOSE_UNSPECIFIED
	}
}
