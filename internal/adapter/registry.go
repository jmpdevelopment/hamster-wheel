package adapter

import (
	"fmt"
	"sync"
)

// Registry manages registered job source adapters.
// It is safe for concurrent use (multiple goroutines polling different adapters).
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

// NewRegistry creates an empty adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register adds an adapter to the registry.
// Returns an error if an adapter with the same name is already registered.
func (r *Registry) Register(adapter Adapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := adapter.Name()
	if _, exists := r.adapters[name]; exists {
		return fmt.Errorf("adapter %q is already registered", name)
	}

	r.adapters[name] = adapter
	return nil
}

// Get retrieves an adapter by name.
// Returns the adapter and true if found, or nil and false if not.
func (r *Registry) Get(name string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.adapters[name]
	return a, ok
}

// List returns all registered adapters.
// The order is not guaranteed.
func (r *Registry) List() []Adapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Adapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		result = append(result, a)
	}
	return result
}

// Names returns the names of all registered adapters.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}
