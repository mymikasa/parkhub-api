package registry

import (
	"log/slog"
	"sync"

	"google.golang.org/grpc"
)

type ServiceRegistration struct {
	Name     string
	Register func(s *grpc.Server)
}

type Registry struct {
	mu       sync.Mutex
	services []ServiceRegistration
}

func New() *Registry {
	return &Registry{}
}

func (r *Registry) MustRegister(name string, fn func(s *grpc.Server)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, svc := range r.services {
		if svc.Name == name {
			slog.Error("duplicate service registration", slog.String("service", name))
			panic("duplicate service registration: " + name)
		}
	}
	r.services = append(r.services, ServiceRegistration{Name: name, Register: fn})
}

func (r *Registry) RegisterAll(s *grpc.Server) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, svc := range r.services {
		slog.Info("registering gRPC service", slog.String("service", svc.Name))
		svc.Register(s)
	}
}
