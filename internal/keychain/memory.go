package keychain

import "sync"

// MemoryStore implements Store using an in-memory map.
// Used in tests where no OS keychain is available.
type MemoryStore struct {
	mu      sync.RWMutex
	secrets map[string]string
}

// NewMemoryStore creates an in-memory Store for testing.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		secrets: make(map[string]string),
	}
}

func (s *MemoryStore) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.secrets[key], nil
}

func (s *MemoryStore) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[key] = value
	return nil
}

func (s *MemoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.secrets, key)
	return nil
}
