package domain

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/parkhub/api/internal/domains/sms/errs"
)

type SmsPurpose string

const (
	SmsPurposeLogin         SmsPurpose = "login"
	SmsPurposeRegister      SmsPurpose = "register"
	SmsPurposeResetPassword SmsPurpose = "reset_password"
)

type SmsCode struct {
	ID        string
	Phone     string
	Code      string
	Purpose   SmsPurpose
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

func NewSmsCode(phone string, purpose SmsPurpose, ttl time.Duration) (*SmsCode, error) {
	if phone == "" {
		return nil, errs.ErrInvalidPhone
	}
	if !isValidPurpose(purpose) {
		return nil, errs.ErrInvalidPurpose
	}

	code, err := generateCode(6)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &SmsCode{
		Phone:     phone,
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: now.Add(ttl),
		Used:      false,
		CreatedAt: now,
	}, nil
}

func (s *SmsCode) IsExpired(now time.Time) bool {
	return now.After(s.ExpiresAt)
}

func (s *SmsCode) Verify(input string) error {
	if s.Used {
		return errs.ErrCodeUsed
	}
	if s.IsExpired(time.Now()) {
		return errs.ErrCodeExpired
	}
	if s.Code != input {
		return errs.ErrCodeMismatch
	}
	return nil
}

func (s *SmsCode) MarkUsed() {
	s.Used = true
}

func isValidPurpose(p SmsPurpose) bool {
	switch p {
	case SmsPurposeLogin, SmsPurposeRegister, SmsPurposeResetPassword:
		return true
	default:
		return false
	}
}

func generateCode(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = byte('0' + n.Int64())
	}
	return string(result), nil
}
