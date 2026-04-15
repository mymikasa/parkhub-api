package errs

import "errors"

var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantAlreadyExists = errors.New("tenant already exists")
	ErrTenantInvalidStatus = errors.New("invalid tenant status transition")

	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserInvalidStatus = errors.New("invalid user status transition")
	ErrUserFrozen        = errors.New("user is frozen")
	ErrPasswordIncorrect = errors.New("incorrect password")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrInvalidRole       = errors.New("invalid user role")

	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrRefreshTokenInvalid = errors.New("refresh token invalid or expired")
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
)
