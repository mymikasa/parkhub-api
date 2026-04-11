package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestCheck_OverallServing(t *testing.T) {
	s := NewServer()
	resp, err := s.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
}

func TestCheck_EmptyServiceOverall(t *testing.T) {
	s := NewServer()
	resp, err := s.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: ""})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
}

func TestCheck_NamedService(t *testing.T) {
	s := NewServer()
	s.SetServing("parkhub.identity.v1.TenantService", grpc_health_v1.HealthCheckResponse_SERVING)

	resp, err := s.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: "parkhub.identity.v1.TenantService"})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
}

func TestCheck_UnknownService(t *testing.T) {
	s := NewServer()
	_, err := s.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: "nonexistent.Service"})
	assert.Error(t, err)
}

func TestSetNotServing(t *testing.T) {
	s := NewServer()
	s.SetServing("parkhub.identity.v1.TenantService", grpc_health_v1.HealthCheckResponse_SERVING)

	s.SetNotServing()

	resp, err := s.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: ""})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, resp.Status)

	resp, err = s.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: "parkhub.identity.v1.TenantService"})
	require.NoError(t, err)
	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, resp.Status)
}

func TestWatch_SendsStatus(t *testing.T) {
	s := NewServer()
	s.SetServing("parkhub.identity.v1.TenantService", grpc_health_v1.HealthCheckResponse_SERVING)

	fakeStream := &watchStream{respCh: make(chan *grpc_health_v1.HealthCheckResponse, 1)}
	err := s.Watch(&grpc_health_v1.HealthCheckRequest{Service: "parkhub.identity.v1.TenantService"}, fakeStream)
	require.NoError(t, err)

	select {
	case resp := <-fakeStream.respCh:
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
	default:
		t.Fatal("expected Watch to send a response")
	}
}

type watchStream struct {
	grpc_health_v1.Health_WatchServer
	respCh chan *grpc_health_v1.HealthCheckResponse
}

func (w *watchStream) Send(resp *grpc_health_v1.HealthCheckResponse) error {
	w.respCh <- resp
	return nil
}
