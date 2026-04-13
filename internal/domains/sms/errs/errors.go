package errs

import "errors"

var (
	ErrCodeNotFound   = errors.New("sms code not found")
	ErrCodeExpired    = errors.New("sms code expired")
	ErrCodeUsed       = errors.New("sms code already used")
	ErrCodeMismatch   = errors.New("sms code mismatch")
	ErrPhoneRateLimit = errors.New("phone rate limit exceeded")
	ErrInvalidPhone   = errors.New("invalid phone number")
	ErrInvalidPurpose = errors.New("invalid sms purpose")
)
