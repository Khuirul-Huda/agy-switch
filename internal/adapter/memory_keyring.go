package adapter

import (
	"sync"

	"github.com/khuirulhuda/agy-switch/internal/port"
)

// MemoryKeyring is an in-memory implementation of port.KeyringStorage used for hermetic testing.
type MemoryKeyring struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewMemoryKeyring constructs a new empty MemoryKeyring.
func NewMemoryKeyring() *MemoryKeyring {
	return &MemoryKeyring{
		data: make(map[string]string),
	}
}

// Get returns the stored secret or port.ErrNotFound.
func (m *MemoryKeyring) Get(account string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	secret, found := m.data[account]
	if !found {
		return "", port.ErrNotFound
	}
	return secret, nil
}

// Set stores the secret in memory.
func (m *MemoryKeyring) Set(account string, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[account] = secret
	return nil
}

// Delete removes the secret from memory.
func (m *MemoryKeyring) Delete(account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, found := m.data[account]; !found {
		return port.ErrNotFound
	}
	delete(m.data, account)
	return nil
}
