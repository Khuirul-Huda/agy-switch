package adapter_test

import (
	"strings"
	"testing"

	"github.com/khuirulhuda/agy-switch/internal/adapter"
)

func TestPromptPasswordUsesInjectedReader(t *testing.T) {
	ui := adapter.NewUIWithReader(strings.NewReader("ignored\n"))
	ui.SetPasswordReader(func() ([]byte, error) {
		return []byte("secret"), nil
	})
	password, err := ui.PromptPassword("Password: ")
	if err != nil {
		t.Fatal(err)
	}
	if password != "secret" {
		t.Fatalf("expected injected password, got %q", password)
	}
}
