package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/khuirulhuda/agy-switch/internal/domain"
	"github.com/khuirulhuda/agy-switch/internal/port"
)

const (
	// ActiveAccountKey is the exact account name read by agy in the OS keyring.
	ActiveAccountKey = "antigravity"

	// ProfilePrefix is the namespace prefix used to store saved profile tokens.
	ProfilePrefix = "antigravity:profile:"

	// RegistryAccountKey stores the list of registered profile names in the keyring.
	RegistryAccountKey = "antigravity:_profiles"
)

var (
	// ErrNoActiveSession indicates that agy has not logged in on this system.
	ErrNoActiveSession = errors.New("no active agy session found in keyring (run agy to sign in first)")

	// ErrProfileNotFound indicates that the requested profile name does not exist.
	ErrProfileNotFound = errors.New("profile does not exist")
)

// ProfileService encapsulates all multi-profile business use cases.
type ProfileService struct {
	storage port.KeyringStorage
}

// NewProfileService creates a new ProfileService instance with injected storage.
func NewProfileService(storage port.KeyringStorage) *ProfileService {
	return &ProfileService{
		storage: storage,
	}
}

// CurrentProfile retrieves the profile currently set as the active agy account.
func (s *ProfileService) CurrentProfile() (*domain.Profile, error) {
	rawToken, err := s.storage.Get(ActiveAccountKey)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNoActiveSession
		}
		return nil, fmt.Errorf("failed to retrieve active token: %w", err)
	}

	payload, err := domain.ParseTokenPayload(rawToken)
	if err != nil {
		return nil, fmt.Errorf("corrupted active token data: %w", err)
	}

	return &domain.Profile{
		Name:    "active",
		Email:   payload.IDTokenEmail,
		RawData: rawToken,
	}, nil
}

// SaveProfile saves the currently active agy token into a named profile and updates the registry.
func (s *ProfileService) SaveProfile(name string) (*domain.Profile, error) {
	if err := domain.ValidateProfileName(name); err != nil {
		return nil, err
	}

	rawToken, err := s.storage.Get(ActiveAccountKey)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNoActiveSession
		}
		return nil, fmt.Errorf("failed to retrieve active token: %w", err)
	}

	payload, err := domain.ParseTokenPayload(rawToken)
	if err != nil {
		return nil, fmt.Errorf("corrupted active token data: %w", err)
	}

	accountKey := formatProfileKey(name)
	if err := s.storage.Set(accountKey, rawToken); err != nil {
		return nil, fmt.Errorf("failed to save profile to keyring: %w", err)
	}

	if err := s.addToRegistry(name); err != nil {
		return nil, fmt.Errorf("failed to register profile name: %w", err)
	}

	return &domain.Profile{
		Name:    name,
		Email:   payload.IDTokenEmail,
		RawData: rawToken,
	}, nil
}

// SwitchProfile activates a previously saved profile by copying its token to the active key.
func (s *ProfileService) SwitchProfile(name string) (*domain.Profile, error) {
	if err := domain.ValidateProfileName(name); err != nil {
		return nil, err
	}

	accountKey := formatProfileKey(name)
	rawToken, err := s.storage.Get(accountKey)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, fmt.Errorf("%w: '%s'", ErrProfileNotFound, name)
		}
		return nil, fmt.Errorf("failed to load profile token: %w", err)
	}

	payload, err := domain.ParseTokenPayload(rawToken)
	if err != nil {
		return nil, fmt.Errorf("corrupted profile token: %w", err)
	}

	if err := s.storage.Set(ActiveAccountKey, rawToken); err != nil {
		return nil, fmt.Errorf("failed to activate profile token: %w", err)
	}

	return &domain.Profile{
		Name:    name,
		Email:   payload.IDTokenEmail,
		RawData: rawToken,
	}, nil
}

// ListProfiles retrieves all stored profiles recorded in the registry.
func (s *ProfileService) ListProfiles() ([]domain.Profile, error) {
	names, err := s.loadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to read profile registry: %w", err)
	}

	var profiles []domain.Profile
	for _, name := range names {
		accountKey := formatProfileKey(name)
		rawToken, err := s.storage.Get(accountKey)
		if err != nil {
			continue
		}

		payload, err := domain.ParseTokenPayload(rawToken)
		if err != nil {
			continue
		}

		profiles = append(profiles, domain.Profile{
			Name:    name,
			Email:   payload.IDTokenEmail,
			RawData: rawToken,
		})
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// DeleteProfile removes a saved profile from keyring and removes it from the registry.
func (s *ProfileService) DeleteProfile(name string) error {
	if err := domain.ValidateProfileName(name); err != nil {
		return err
	}

	accountKey := formatProfileKey(name)
	if err := s.storage.Delete(accountKey); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return fmt.Errorf("%w: '%s'", ErrProfileNotFound, name)
		}
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	return s.removeFromRegistry(name)
}

// LogoutActive removes the active agy account token without touching saved profiles.
func (s *ProfileService) LogoutActive() error {
	if err := s.storage.Delete(ActiveAccountKey); err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("failed to clear active session: %w", err)
	}
	return nil
}

func (s *ProfileService) loadRegistry() ([]string, error) {
	raw, err := s.storage.Get(RegistryAccountKey)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return []string{}, nil
		}
		return nil, err
	}

	var names []string
	if err := json.Unmarshal([]byte(raw), &names); err != nil {
		return []string{}, nil
	}
	return names, nil
}

func (s *ProfileService) saveRegistry(names []string) error {
	data, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return s.storage.Set(RegistryAccountKey, string(data))
}

func (s *ProfileService) addToRegistry(name string) error {
	names, err := s.loadRegistry()
	if err != nil {
		return err
	}

	for _, existing := range names {
		if existing == name {
			return nil
		}
	}

	names = append(names, name)
	return s.saveRegistry(names)
}

func (s *ProfileService) removeFromRegistry(name string) error {
	names, err := s.loadRegistry()
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(names))
	for _, existing := range names {
		if existing != name {
			filtered = append(filtered, existing)
		}
	}

	return s.saveRegistry(filtered)
}

func formatProfileKey(profileName string) string {
	return ProfilePrefix + profileName
}
