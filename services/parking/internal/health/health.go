package health

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	grpc_health_v1.UnimplementedHealthServer
	serving atomic.Bool
}

func NewServer() *Server {
	s := &Server{}
	s.serving.Store(true)
	return s
}

func (s *Server) Check(_ context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	status := grpc_health_v1.HealthCheckResponse_SERVING
	if !s.serving.Load() {
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}
	return &grpc_health_v1.HealthCheckResponse{Status: status}, nil
}

func (s *Server) SetNotServing() {
	s.serving.Store(false)
}

func RegisterGRPC(s *grpc.Server, h *Server) {
	grpc_health_v1.RegisterHealthServer(s, h)
}
