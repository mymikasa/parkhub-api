package registry

import (
	"fmt"
	"sync"

	"google.golang.org/grpc"
)

type Registry struct {
	mu       sync.Mutex
	services map[string]func(*grpc.Server)
}

func New() *Registry {
	return &Registry{
		services: make(map[string]func(*grpc.Server)),
	}
}

func (r *Registry) MustRegister(name string, register func(*grpc.Server)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.services[name]; exists {
		panic(fmt.Sprintf("service %q already registered", name))
	}
	r.services[name] = register
}

func (r *Registry) RegisterAll(s *grpc.Server) {
	r.mu.Lock()
	services := make(map[string]func(*grpc.Server), len(r.services))
	for k, v := range r.services {
		services[k] = v
	}
	r.mu.Unlock()

	for _, register := range services {
		register(s)
	}
}
