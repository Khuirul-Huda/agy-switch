package main

import (
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

func TestHandleNew_DoesNotSaveWhenLoginFails(t *testing.T) {
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
	if _, err := svc.CurrentProfile(); !errors.Is(err, service.ErrNoActiveSession) {
		t.Fatalf("expected active session to be cleared after failed login, got: %v", err)
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
