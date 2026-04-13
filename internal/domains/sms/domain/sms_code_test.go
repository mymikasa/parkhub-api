package domain

import (
	"testing"
	"time"

	"github.com/parkhub/api/internal/domains/sms/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSmsCode_Success(t *testing.T) {
	code, err := NewSmsCode("13800138000", SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)

	assert.Equal(t, "13800138000", code.Phone)
	assert.Equal(t, SmsPurposeLogin, code.Purpose)
	assert.Len(t, code.Code, 6)
	assert.False(t, code.Used)
	assert.WithinDuration(t, time.Now().Add(5*time.Minute), code.ExpiresAt, time.Second)
}

func TestNewSmsCode_EmptyPhone(t *testing.T) {
	_, err := NewSmsCode("", SmsPurposeLogin, 5*time.Minute)
	assert.ErrorIs(t, err, errs.ErrInvalidPhone)
}

func TestNewSmsCode_InvalidPurpose(t *testing.T) {
	_, err := NewSmsCode("13800138000", SmsPurpose("unknown"), 5*time.Minute)
	assert.ErrorIs(t, err, errs.ErrInvalidPurpose)
}

func TestNewSmsCode_DifferentPurposes(t *testing.T) {
	tests := []struct {
		name    string
		purpose SmsPurpose
	}{
		{"login", SmsPurposeLogin},
		{"register", SmsPurposeRegister},
		{"reset_password", SmsPurposeResetPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := NewSmsCode("13800138000", tt.purpose, 5*time.Minute)
			require.NoError(t, err)
			assert.Equal(t, tt.purpose, code.Purpose)
		})
	}
}

func TestNewSmsCode_CodeIsNumeric(t *testing.T) {
	code, err := NewSmsCode("13800138000", SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)
	for _, c := range code.Code {
		assert.True(t, c >= '0' && c <= '9', "code should be numeric, got: %c", c)
	}
}

func TestSmsCode_IsExpired(t *testing.T) {
	code, err := NewSmsCode("13800138000", SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)

	assert.False(t, code.IsExpired(time.Now()))
	assert.True(t, code.IsExpired(time.Now().Add(6*time.Minute)))
}

func TestSmsCode_Verify_Success(t *testing.T) {
	code, err := NewSmsCode("13800138000", SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)

	err = code.Verify(code.Code)
	assert.NoError(t, err)
}

func TestSmsCode_Verify_Mismatch(t *testing.T) {
	code, err := NewSmsCode("13800138000", SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)

	err = code.Verify("000000")
	assert.ErrorIs(t, err, errs.ErrCodeMismatch)
}

func TestSmsCode_Verify_Expired(t *testing.T) {
	code := &SmsCode{
		Phone:     "13800138000",
		Code:      "123456",
		Purpose:   SmsPurposeLogin,
		ExpiresAt: time.Now().Add(-1 * time.Minute),
		Used:      false,
	}

	err := code.Verify("123456")
	assert.ErrorIs(t, err, errs.ErrCodeExpired)
}

func TestSmsCode_Verify_Used(t *testing.T) {
	code, err := NewSmsCode("13800138000", SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)

	code.MarkUsed()
	err = code.Verify(code.Code)
	assert.ErrorIs(t, err, errs.ErrCodeUsed)
}

func TestSmsCode_MarkUsed(t *testing.T) {
	code, err := NewSmsCode("13800138000", SmsPurposeLogin, 5*time.Minute)
	require.NoError(t, err)

	assert.False(t, code.Used)
	code.MarkUsed()
	assert.True(t, code.Used)
}
