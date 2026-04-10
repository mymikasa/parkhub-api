package errs

import "errors"

var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantAlreadyExists = errors.New("tenant already exists")
	ErrTenantInvalidStatus = errors.New("invalid tenant status transition")
)
