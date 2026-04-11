package health

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type Server struct {
	grpc_health_v1.UnimplementedHealthServer
	mu     sync.RWMutex
	status map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
}

func NewServer() *Server {
	return &Server{
		status: map[string]grpc_health_v1.HealthCheckResponse_ServingStatus{
			"": grpc_health_v1.HealthCheckResponse_SERVING,
		},
	}
}

func (s *Server) Check(_ context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, ok := s.status[req.Service]
	if !ok {
		return nil, status.Error(codes.NotFound, "unknown service")
	}
	return &grpc_health_v1.HealthCheckResponse{Status: st}, nil
}

func (s *Server) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	s.mu.RLock()
	st, ok := s.status[req.Service]
	s.mu.RUnlock()
	if !ok {
		return status.Error(codes.NotFound, "unknown service")
	}
	return stream.Send(&grpc_health_v1.HealthCheckResponse{Status: st})
}

func (s *Server) SetServing(service string, st grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status[service] = st
}

func (s *Server) SetNotServing() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.status {
		s.status[k] = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}
}

func RegisterGRPC(srv grpc.ServiceRegistrar, s *Server) {
	grpc_health_v1.RegisterHealthServer(srv, s)
}
