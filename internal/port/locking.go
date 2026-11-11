package port

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

// ProcessLocker serializes mutations that span multiple keyring accounts.
type ProcessLocker interface {
	Lock() error
	Unlock() error
}

// NoopLocker is useful for hermetic service tests.
type NoopLocker struct{}

func (NoopLocker) Lock() error   { return nil }
func (NoopLocker) Unlock() error { return nil }

// FileLocker uses an OS-level advisory lock and works on Linux, macOS, and Windows.
type FileLocker struct {
	lock *flock.Flock
}

// NewDefaultProcessLocker creates a per-user lock file in the platform config directory.
func NewDefaultProcessLocker() (*FileLocker, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user config directory: %w", err)
	}
	dir := filepath.Join(configDir, "agy-switch")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	return NewProcessLocker(filepath.Join(dir, "profiles.lock")), nil
}

// NewProcessLocker creates a lock for a caller-supplied path, primarily for tests.
func NewProcessLocker(path string) *FileLocker {
	return &FileLocker{lock: flock.New(path)}
}

func (l *FileLocker) Lock() error {
	if l == nil || l.lock == nil {
		return nil
	}
	return l.lock.Lock()
}

func (l *FileLocker) Unlock() error {
	if l == nil || l.lock == nil {
		return nil
	}
	return l.lock.Unlock()
}
