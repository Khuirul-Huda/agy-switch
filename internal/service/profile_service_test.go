package service_test

import (
	"encoding/json"
	"errors"
	"strings"
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

func TestProfileExists(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)

	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	if _, err := svc.SaveProfile("personal"); err != nil {
		t.Fatalf("failed to seed profile: %v", err)
	}

	exists, err := svc.ProfileExists("personal")
	if err != nil {
		t.Fatalf("unexpected error checking existing profile: %v", err)
	}
	if !exists {
		t.Fatal("expected profile to exist")
	}

	exists, err = svc.ProfileExists("work")
	if err != nil {
		t.Fatalf("unexpected error checking missing profile: %v", err)
	}
	if exists {
		t.Fatal("expected profile not to exist")
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

func TestSaveProfileRejectsOverwriteAndForceAllowsIt(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)
	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	if _, err := svc.SaveProfile("personal"); err != nil {
		t.Fatal(err)
	}
	_ = storage.Set(service.ActiveAccountKey, secondRawToken)
	if _, err := svc.SaveProfile("personal"); !errors.Is(err, service.ErrProfileExists) {
		t.Fatalf("expected duplicate protection, got %v", err)
	}
	if _, err := svc.SaveProfileForce("personal"); err != nil {
		t.Fatal(err)
	}
	profiles, err := svc.ListProfiles()
	if err != nil || len(profiles) != 1 || profiles[0].Email != "work@company.com" {
		t.Fatalf("force did not replace profile: %+v, %v", profiles, err)
	}
}

func TestRenameProfileMovesTokenAndMetadata(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)
	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	if _, err := svc.SaveProfile("old"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RenameProfile("old", "new"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SwitchProfile("old"); !errors.Is(err, service.ErrProfileNotFound) {
		t.Fatalf("old profile still exists: %v", err)
	}
	profile, err := svc.SwitchProfile("new")
	if err != nil || profile.Email != "test@example.com" {
		t.Fatalf("renamed profile unavailable: %v", err)
	}
}

func TestListProfilesMarksOnlyMatchingActiveToken(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)
	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	_, _ = svc.SaveProfile("personal")
	_ = storage.Set(service.ActiveAccountKey, `{"id_token_email":"unmatched@example.com"}`)
	profiles, err := svc.ListProfiles()
	if err != nil || len(profiles) != 1 {
		t.Fatalf("unexpected profiles: %+v, %v", profiles, err)
	}
	if profiles[0].Active {
		t.Fatal("unmatched active token was marked active")
	}
}

func TestLegacyRegistryMigratesToMetadata(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)
	_ = storage.Set(service.RegistryAccountKey, `["legacy"]`)
	_ = storage.Set(service.ProfilePrefix+"legacy", sampleRawToken)
	profiles, err := svc.ListProfiles()
	if err != nil || len(profiles) != 1 {
		t.Fatalf("legacy registry was not loaded: %+v, %v", profiles, err)
	}
	if profiles[0].CreatedAt.IsZero() {
		t.Fatal("legacy profile did not receive created_at metadata")
	}
	var migrated map[string]any
	raw, _ := storage.Get(service.RegistryAccountKey)
	if json.Unmarshal([]byte(raw), &migrated) != nil || migrated["version"] == nil {
		t.Fatalf("registry was not migrated: %s", raw)
	}
}

func TestDoctorReportsBrokenProfileWithoutExposingToken(t *testing.T) {
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)
	_ = storage.Set(service.RegistryAccountKey, `{"version":1,"profiles":{"broken":{"created_at":"2024-01-01T00:00:00Z"}}}`)
	_ = storage.Set(service.ProfilePrefix+"broken", "not-json")

	report := svc.Doctor(func() (string, error) { return "/usr/bin/agy", nil })
	if report.OK {
		t.Fatal("expected doctor to report a broken profile")
	}
	for _, diagnostic := range report.Diagnostics {
		if strings.Contains(diagnostic.Message, "not-json") {
			t.Fatal("doctor leaked token contents")
		}
	}
}
