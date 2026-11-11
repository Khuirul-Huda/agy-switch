package port_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/khuirulhuda/agy-switch/internal/port"
)

func TestFileLockerContendsAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.lock")
	first := port.NewProcessLocker(path)
	second := port.NewProcessLocker(path)
	if err := first.Lock(); err != nil {
		t.Fatal(err)
	}
	defer first.Unlock()

	acquired := make(chan struct{})
	go func() {
		if err := second.Lock(); err == nil {
			close(acquired)
			_ = second.Unlock()
		}
	}()
	select {
	case <-acquired:
		t.Fatal("second process lock acquired while first lock was held")
	case <-time.After(100 * time.Millisecond):
	}
}
