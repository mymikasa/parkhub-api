package gateway

import (
	"context"
	"log/slog"
)

type MockSmsGateway struct {
	Calls []CallRecord
	Err   error
}

func NewMockSmsGateway() *MockSmsGateway {
	return &MockSmsGateway{}
}

func (m *MockSmsGateway) Send(_ context.Context, phone, code string, purpose string) error {
	m.Calls = append(m.Calls, CallRecord{
		Phone:   phone,
		Code:    code,
		Purpose: purpose,
	})
	slog.Info("[MockSMS] verification code sent", "phone", phone, "code", code, "purpose", purpose)
	return m.Err
}
