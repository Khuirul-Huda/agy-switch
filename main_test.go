package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/khuirulhuda/agy-switch/internal/adapter"
	"github.com/khuirulhuda/agy-switch/internal/service"
)

func TestHandleNew_CreatesProfileAfterSuccessfulLogin(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	_ = storage.Set(service.ActiveAccountKey, `{"id_token_email":"old@example.com"}`)
	svc := service.NewProfileService(storage)
	ui := adapter.NewUIWithReader(strings.NewReader("work\n"))

	const newToken = `{"id_token_email":"work@example.com"}`
	runAgy := func() error {
		return storage.Set(service.ActiveAccountKey, newToken)
	}

	handleNew(svc, ui, runAgy)

	profile, err := svc.SwitchProfile("work")
	if err != nil {
		t.Fatalf("expected new profile to be saved: %v", err)
	}
	if profile.Email != "work@example.com" {
		t.Fatalf("expected work email, got %q", profile.Email)
	}
}

func TestHandleNew_DoesNotLogoutExistingProfile(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	_ = storage.Set(service.ActiveAccountKey, `{"id_token_email":"active@example.com"}`)
	svc := service.NewProfileService(storage)
	if _, err := svc.SaveProfile("work"); err != nil {
		t.Fatalf("failed to seed profile: %v", err)
	}
	_ = storage.Set(service.ActiveAccountKey, `{"id_token_email":"active@example.com"}`)
	ui := adapter.NewUIWithReader(strings.NewReader("work\n"))
	runCalled := false

	handleNew(svc, ui, func() error {
		runCalled = true
		return nil
	})

	if runCalled {
		t.Fatal("expected agy not to run for a duplicate profile")
	}
	active, err := svc.CurrentProfile()
	if err != nil {
		t.Fatalf("expected active session to remain: %v", err)
	}
	if active.Email != "active@example.com" {
		t.Fatalf("expected active session to remain unchanged, got %q", active.Email)
	}
}

func TestHandleNew_RestoresSessionWhenLoginFails(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	_ = storage.Set(service.ActiveAccountKey, `{"id_token_email":"old@example.com"}`)
	svc := service.NewProfileService(storage)
	ui := adapter.NewUIWithReader(strings.NewReader("work\n"))
	loginErr := errors.New("agy unavailable")

	handleNew(svc, ui, func() error {
		return loginErr
	})

	if _, err := svc.SwitchProfile("work"); !errors.Is(err, service.ErrProfileNotFound) {
		t.Fatalf("expected profile not to be saved, got: %v", err)
	}
	active, err := svc.CurrentProfile()
	if err != nil {
		t.Fatalf("expected previous active session to be restored, got: %v", err)
	}
	if active.RawData != `{"id_token_email":"old@example.com"}` {
		t.Fatalf("expected previous active token to be restored, got %q", active.RawData)
	}
}

func TestHandleNew_RejectsInvalidNameBeforeLogout(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	const activeToken = `{"id_token_email":"old@example.com"}`
	_ = storage.Set(service.ActiveAccountKey, activeToken)
	svc := service.NewProfileService(storage)
	ui := adapter.NewUIWithReader(strings.NewReader("invalid name\n"))
	runCalled := false

	handleNew(svc, ui, func() error {
		runCalled = true
		return nil
	})

	if runCalled {
		t.Fatal("expected agy not to run for an invalid profile name")
	}
	active, err := svc.CurrentProfile()
	if err != nil {
		t.Fatalf("expected active session to remain: %v", err)
	}
	if active.RawData != activeToken {
		t.Fatal("expected active token to remain unchanged")
	}
}

func TestHandleNew_NonInteractiveNameAndRetry(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	const previous = `{"id_token_email":"old@example.com"}`
	_ = storage.Set(service.ActiveAccountKey, previous)
	svc := service.NewProfileService(storage)
	ui := adapter.NewUIWithReader(strings.NewReader("y\n"))
	attempts := 0
	handleNewArgs(svc, ui, func() error {
		attempts++
		if attempts == 1 {
			return errors.New("temporary login failure")
		}
		return storage.Set(service.ActiveAccountKey, `{"id_token_email":"new@example.com"}`)
	}, []string{"new-profile"})
	if attempts != 2 {
		t.Fatalf("expected one retry, got %d attempts", attempts)
	}
	profile, err := svc.SwitchProfile("new-profile")
	if err != nil || profile.Email != "new@example.com" {
		t.Fatalf("expected retried profile, got %+v, %v", profile, err)
	}
}

func TestHandleNew_RestoresSessionWhenAgyDoesNotChangeToken(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	const previous = `{"id_token_email":"old@example.com"}`
	_ = storage.Set(service.ActiveAccountKey, previous)
	svc := service.NewProfileService(storage)
	ui := adapter.NewUIWithReader(strings.NewReader("work\n"))

	handleNew(svc, ui, func() error {
		return nil
	})

	if _, err := svc.SwitchProfile("work"); !errors.Is(err, service.ErrProfileNotFound) {
		t.Fatalf("expected profile not to be saved, got: %v", err)
	}
	active, err := svc.CurrentProfile()
	if err != nil {
		t.Fatalf("expected previous active session to be restored, got: %v", err)
	}
	if active.RawData != previous {
		t.Fatalf("expected previous token to be restored, got %q", active.RawData)
	}
}

func TestCompletionScriptIncludesProfiles(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		script := completionScript(shell, []string{"personal", "work"})
		if !strings.Contains(script, "personal") || !strings.Contains(script, "work") {
			t.Fatalf("%s completion omitted profile names: %s", shell, script)
		}
	}
}

func TestHandleCurrentQuietIsSilentOnSuccess(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	_ = storage.Set(service.ActiveAccountKey, `{"id_token_email":"quiet@example.com"}`)
	svc := service.NewProfileService(storage)
	var out, errOut bytes.Buffer
	ui := adapter.NewUIWithWriters(strings.NewReader(""), &out, &errOut)
	handleCurrent(svc, ui, true)
	if out.Len() != 0 || errOut.Len() != 0 {
		t.Fatalf("quiet current produced output: stdout=%q stderr=%q", out.String(), errOut.String())
	}
}

func TestHandleCurrentJSONIsTokenFreeAndStable(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	_ = storage.Set(service.ActiveAccountKey, `{"id_token_email":"json@example.com","token":{"access_token":"secret"}}`)
	svc := service.NewProfileService(storage)
	var out, errOut bytes.Buffer
	ui := adapter.NewUIWithWriters(strings.NewReader(""), &out, &errOut)
	if !handleCurrent(svc, ui, false, true) {
		t.Fatal("expected current --json to succeed")
	}
	if out.String() != "{\"email\":\"json@example.com\"}\n" {
		t.Fatalf("unexpected JSON output: %q", out.String())
	}
	if strings.Contains(out.String(), "secret") || errOut.Len() != 0 {
		t.Fatal("current --json leaked a token or wrote an error")
	}
}

func TestHandleCurrentJSONErrorHasStableCode(t *testing.T) {
	svc := service.NewProfileService(adapter.NewMemoryKeyring())
	var out, errOut bytes.Buffer
	ui := adapter.NewUIWithWriters(strings.NewReader(""), &out, &errOut)
	if handleCurrent(svc, ui, false, true) {
		t.Fatal("expected current --json to fail without an active session")
	}
	if out.String() != "{\"error\":{\"code\":\"no_active_session\",\"message\":\"no active agy session found\"}}\n" {
		t.Fatalf("unexpected JSON error: %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatal("JSON error should be machine-readable on stdout only")
	}
}

func TestReadPasswordPrefersEnvironment(t *testing.T) {
	t.Setenv(passwordEnv, "ci-secret")
	ui := adapter.NewUIWithReader(strings.NewReader("prompt-secret\n"))
	password, err := readPassword(ui, "Password: ")
	if err != nil {
		t.Fatal(err)
	}
	if password != "ci-secret" {
		t.Fatalf("expected environment password, got %q", password)
	}
}
