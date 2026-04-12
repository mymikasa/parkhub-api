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
	mu       sync.RWMutex
	status   map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
	watchers map[string][]chan grpc_health_v1.HealthCheckResponse_ServingStatus
}

func NewServer() *Server {
	return &Server{
		status: map[string]grpc_health_v1.HealthCheckResponse_ServingStatus{
			"": grpc_health_v1.HealthCheckResponse_SERVING,
		},
		watchers: make(map[string][]chan grpc_health_v1.HealthCheckResponse_ServingStatus),
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
	ch := s.addWatcher(req.Service)
	defer s.removeWatcher(req.Service, ch)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case st := <-ch:
			if err := stream.Send(&grpc_health_v1.HealthCheckResponse{Status: st}); err != nil {
				return err
			}
		}
	}
}

func (s *Server) addWatcher(service string) chan grpc_health_v1.HealthCheckResponse_ServingStatus {
	ch := make(chan grpc_health_v1.HealthCheckResponse_ServingStatus, 1)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.watchers[service] = append(s.watchers[service], ch)

	if st, ok := s.status[service]; ok {
		ch <- st
	} else {
		ch <- grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN
	}
	return ch
}

func (s *Server) removeWatcher(service string, ch chan grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	watchers := s.watchers[service]
	for i, w := range watchers {
		if w == ch {
			s.watchers[service] = append(watchers[:i], watchers[i+1:]...)
			break
		}
	}
}

func (s *Server) SetServing(service string, st grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status[service] = st
	s.notifyWatchers(service, st)
}

func (s *Server) SetNotServing() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.status {
		s.status[k] = grpc_health_v1.HealthCheckResponse_NOT_SERVING
		s.notifyWatchers(k, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}
}

func (s *Server) notifyWatchers(service string, st grpc_health_v1.HealthCheckResponse_ServingStatus) {
	for _, ch := range s.watchers[service] {
		select {
		case ch <- st:
		default:
		}
	}
	if service != "" {
		for _, ch := range s.watchers[""] {
			select {
			case ch <- st:
			default:
			}
		}
	}
}

func RegisterGRPC(srv grpc.ServiceRegistrar, s *Server) {
	grpc_health_v1.RegisterHealthServer(srv, s)
}
