package gateway

import "context"

type CallRecord struct {
	Phone   string
	Code    string
	Purpose string
}

type SmsGateway interface {
	Send(ctx context.Context, phone, code string, purpose string) error
}
