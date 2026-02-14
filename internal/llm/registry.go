package llm

import (
	"fmt"
	"sort"
	"sync"
)

// Registry stores match providers by name.
// It is safe for concurrent use by worker and settings/runtime code.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register inserts a provider. Provider names must be unique.
func (r *Registry) Register(provider Provider) error {
	if provider == nil {
		return fmt.Errorf("provider is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %q is already registered", name)
	}
	r.providers[name] = provider
	return nil
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	return p, ok
}

// Names returns registered provider names in sorted order for deterministic UI/tests.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
