package gateway

import "context"

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
	return m.Err
}
