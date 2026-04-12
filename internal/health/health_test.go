package health

import (
	"context"
	"testing"
	"time"

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

func TestWatch_SendsInitialStatus(t *testing.T) {
	s := NewServer()
	s.SetServing("parkhub.identity.v1.TenantService", grpc_health_v1.HealthCheckResponse_SERVING)

	fakeStream := &watchStream{
		ctx:    context.Background(),
		respCh: make(chan *grpc_health_v1.HealthCheckResponse, 4),
	}
	go func() {
		_ = s.Watch(&grpc_health_v1.HealthCheckRequest{Service: "parkhub.identity.v1.TenantService"}, fakeStream)
	}()

	select {
	case resp := <-fakeStream.respCh:
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
	case <-time.After(time.Second):
		t.Fatal("expected Watch to send initial status")
	}
}

func TestWatch_UnknownService_SendsServiceUnknown(t *testing.T) {
	s := NewServer()

	fakeStream := &watchStream{
		ctx:    context.Background(),
		respCh: make(chan *grpc_health_v1.HealthCheckResponse, 4),
	}
	go func() {
		_ = s.Watch(&grpc_health_v1.HealthCheckRequest{Service: "nonexistent.Service"}, fakeStream)
	}()

	select {
	case resp := <-fakeStream.respCh:
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN, resp.Status)
	case <-time.After(time.Second):
		t.Fatal("expected Watch to send SERVICE_UNKNOWN")
	}
}

func TestWatch_ReceivesStatusUpdate(t *testing.T) {
	s := NewServer()

	fakeStream := &watchStream{
		ctx:    context.Background(),
		respCh: make(chan *grpc_health_v1.HealthCheckResponse, 4),
	}
	go func() {
		_ = s.Watch(&grpc_health_v1.HealthCheckRequest{Service: ""}, fakeStream)
	}()

	select {
	case resp := <-fakeStream.respCh:
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
	case <-time.After(time.Second):
		t.Fatal("expected initial SERVING")
	}

	s.SetNotServing()

	select {
	case resp := <-fakeStream.respCh:
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, resp.Status)
	case <-time.After(time.Second):
		t.Fatal("expected status update to NOT_SERVING")
	}
}

func TestWatch_ContextCancel(t *testing.T) {
	s := NewServer()
	ctx, cancel := context.WithCancel(context.Background())

	fakeStream := &watchStream{
		ctx:    ctx,
		respCh: make(chan *grpc_health_v1.HealthCheckResponse, 4),
	}
	done := make(chan error, 1)
	go func() {
		done <- s.Watch(&grpc_health_v1.HealthCheckRequest{Service: ""}, fakeStream)
	}()

	<-fakeStream.respCh

	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("expected Watch to return on context cancel")
	}
}

type watchStream struct {
	grpc_health_v1.Health_WatchServer
	ctx    context.Context
	respCh chan *grpc_health_v1.HealthCheckResponse
}

func (w *watchStream) Send(resp *grpc_health_v1.HealthCheckResponse) error {
	w.respCh <- resp
	return nil
}

func (w *watchStream) Context() context.Context {
	return w.ctx
}
