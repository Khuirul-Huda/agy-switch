package adapter

import (
	"errors"
	"fmt"

	"github.com/khuirulhuda/agy-switch/internal/port"
	"github.com/zalando/go-keyring"
)

const (
	// DefaultServiceName is the service identifier used by agy across all supported OSes.
	DefaultServiceName = "gemini"
)

// SystemKeyring is the production adapter that talks directly to the OS credential store
// (Linux DBus Secret Service, macOS Apple Keychain, Windows Credential Manager).
type SystemKeyring struct {
	serviceName string
}

// NewSystemKeyring initializes a SystemKeyring with the specified service name.
func NewSystemKeyring(serviceName string) *SystemKeyring {
	if serviceName == "" {
		serviceName = DefaultServiceName
	}
	return &SystemKeyring{
		serviceName: serviceName,
	}
}

// Get retrieves a secret from the underlying OS Keyring.
func (k *SystemKeyring) Get(account string) (string, error) {
	secret, err := keyring.Get(k.serviceName, account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", port.ErrNotFound
		}
		return "", fmt.Errorf("system keyring get failed: %w", err)
	}
	return secret, nil
}

// Set stores a secret in the underlying OS Keyring.
func (k *SystemKeyring) Set(account string, secret string) error {
	if err := keyring.Set(k.serviceName, account, secret); err != nil {
		return fmt.Errorf("system keyring set failed: %w", err)
	}
	return nil
}

// Delete removes a secret from the underlying OS Keyring.
func (k *SystemKeyring) Delete(account string) error {
	err := keyring.Delete(k.serviceName, account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return port.ErrNotFound
		}
		return fmt.Errorf("system keyring delete failed: %w", err)
	}
	return nil
}
