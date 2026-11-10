package service_test

import (
	"errors"
	"testing"

	"github.com/khuirulhuda/agy-switch/internal/adapter"
	"github.com/khuirulhuda/agy-switch/internal/domain"
	"github.com/khuirulhuda/agy-switch/internal/service"
)

const sampleRawToken = `{"auth_method":"consumer","id_token_email":"test@example.com","token":{"access_token":"abc123xyz"}}`
const secondRawToken = `{"auth_method":"consumer","id_token_email":"work@company.com","token":{"access_token":"def456uvw"}}`

func TestCurrentProfile_NoActiveSession(t *testing.T) {
	// Arrange: Clean in-memory storage (zero OS keyring touch)
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)

	// Act
	profile, err := svc.CurrentProfile()

	// Assert
	if !errors.Is(err, service.ErrNoActiveSession) {
		t.Fatalf("expected ErrNoActiveSession, got: %v", err)
	}
	if profile != nil {
		t.Fatalf("expected nil profile, got: %+v", profile)
	}
}

func TestCurrentProfile_ActiveSessionFound(t *testing.T) {
	// Arrange
	storage := adapter.NewMemoryKeyring()
	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	svc := service.NewProfileService(storage)

	// Act
	profile, err := svc.CurrentProfile()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", profile.Email)
	}
}

func TestSaveProfile_Success(t *testing.T) {
	// Arrange
	storage := adapter.NewMemoryKeyring()
	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	svc := service.NewProfileService(storage)

	// Act
	saved, err := svc.SaveProfile("personal")

	// Assert
	if err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}
	if saved.Name != "personal" {
		t.Errorf("expected profile name 'personal', got '%s'", saved.Name)
	}
	if saved.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", saved.Email)
	}

	// Verify in storage
	stored, err := storage.Get(service.ProfilePrefix + "personal")
	if err != nil {
		t.Fatalf("expected key to exist in storage, got error: %v", err)
	}
	if stored != sampleRawToken {
		t.Errorf("stored payload mismatch")
	}
}

func TestSaveProfile_InvalidName(t *testing.T) {
	// Arrange
	storage := adapter.NewMemoryKeyring()
	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	svc := service.NewProfileService(storage)

	// Act
	_, err := svc.SaveProfile("invalid name with space")

	// Assert
	if !errors.Is(err, domain.ErrInvalidProfileName) {
		t.Fatalf("expected ErrInvalidProfileName, got: %v", err)
	}
}

func TestSaveProfile_NoActiveSession(t *testing.T) {
	// Arrange
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)

	// Act
	_, err := svc.SaveProfile("work")

	// Assert
	if !errors.Is(err, service.ErrNoActiveSession) {
		t.Fatalf("expected ErrNoActiveSession, got: %v", err)
	}
}

func TestSwitchProfile_Success(t *testing.T) {
	// Arrange: Save work profile first, then activate personal, then switch to work
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)

	_ = storage.Set(service.ActiveAccountKey, secondRawToken)
	_, _ = svc.SaveProfile("work")

	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)

	// Act
	switched, err := svc.SwitchProfile("work")

	// Assert
	if err != nil {
		t.Fatalf("failed to switch profile: %v", err)
	}
	if switched.Name != "work" {
		t.Errorf("expected name 'work', got '%s'", switched.Name)
	}
	if switched.Email != "work@company.com" {
		t.Errorf("expected email 'work@company.com', got '%s'", switched.Email)
	}

	// Verify active token in storage now matches work
	activeRaw, err := storage.Get(service.ActiveAccountKey)
	if err != nil {
		t.Fatalf("failed to get active token: %v", err)
	}
	if activeRaw != secondRawToken {
		t.Errorf("active token was not updated correctly")
	}
}

func TestSwitchProfile_NotFound(t *testing.T) {
	// Arrange
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)

	// Act
	_, err := svc.SwitchProfile("non-existent")

	// Assert
	if !errors.Is(err, service.ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got: %v", err)
	}
}

func TestListProfiles_EmptyAndPopulated(t *testing.T) {
	// Arrange
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)

	// Act 1: Empty list
	emptyList, err := svc.ListProfiles()
	if err != nil {
		t.Fatalf("unexpected error on empty list: %v", err)
	}
	if len(emptyList) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(emptyList))
	}

	// Act 2: Populated
	_ = storage.Set(service.ActiveAccountKey, secondRawToken)
	_, _ = svc.SaveProfile("work")

	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	_, _ = svc.SaveProfile("personal")

	list, err := svc.ListProfiles()
	if err != nil {
		t.Fatalf("failed to list profiles: %v", err)
	}

	// Assert
	if len(list) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(list))
	}
	// Sorted alphabetically by name
	if list[0].Name != "personal" || list[1].Name != "work" {
		t.Errorf("expected sorted profiles ['personal', 'work'], got ['%s', '%s']", list[0].Name, list[1].Name)
	}
}

func TestDeleteProfile_SuccessAndNotFound(t *testing.T) {
	// Arrange
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)

	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	_, _ = svc.SaveProfile("temporary")

	// Act 1: Delete existing
	err := svc.DeleteProfile("temporary")
	if err != nil {
		t.Fatalf("failed to delete profile: %v", err)
	}

	// Assert 1: Key removed
	_, err = storage.Get(service.ProfilePrefix + "temporary")
	if err == nil {
		t.Errorf("expected key to be deleted")
	}

	// Act 2: Delete again (should return ErrProfileNotFound)
	err = svc.DeleteProfile("temporary")
	if !errors.Is(err, service.ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound on second delete, got: %v", err)
	}
}

func TestLogoutActive_Success(t *testing.T) {
	// Arrange
	storage := adapter.NewMemoryKeyring()
	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	svc := service.NewProfileService(storage)

	// Act
	err := svc.LogoutActive()

	// Assert
	if err != nil {
		t.Fatalf("failed to logout: %v", err)
	}
	_, err = storage.Get(service.ActiveAccountKey)
	if err == nil {
		t.Errorf("expected active account to be deleted from keyring")
	}
}
