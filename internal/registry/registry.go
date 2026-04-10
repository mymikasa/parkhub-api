package registry

import (
	"log"
	"sync"

	"google.golang.org/grpc"
)

// ServiceRegistration holds a service name and its registration function.
type ServiceRegistration struct {
	Name     string
	Register func(s *grpc.Server)
}

// Registry collects gRPC service registrations.
type Registry struct {
	mu       sync.Mutex
	services []ServiceRegistration
}

// New creates a new Registry.
func New() *Registry {
	return &Registry{}
}

// MustRegister adds a service registration. Panics on duplicate names.
func (r *Registry) MustRegister(name string, fn func(s *grpc.Server)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, svc := range r.services {
		if svc.Name == name {
			log.Fatalf("duplicate service registration: %s", name)
		}
	}
	r.services = append(r.services, ServiceRegistration{Name: name, Register: fn})
}

// RegisterAll registers all collected services onto the gRPC server.
func (r *Registry) RegisterAll(s *grpc.Server) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, svc := range r.services {
		log.Printf("registering gRPC service: %s", svc.Name)
		svc.Register(s)
	}
}
