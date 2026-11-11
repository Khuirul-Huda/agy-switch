package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/khuirulhuda/agy-switch/internal/domain"
	"github.com/khuirulhuda/agy-switch/internal/port"
)

const (
	ActiveAccountKey   = "antigravity"
	ProfilePrefix      = "antigravity:profile:"
	RegistryAccountKey = "antigravity:_profiles"
	registryVersion    = 1
)

var (
	ErrNoActiveSession = errors.New("no active agy session found in keyring (run agy to sign in first)")
	ErrProfileNotFound = errors.New("profile does not exist")
	ErrProfileExists   = errors.New("profile already exists")
	ErrInvalidExport   = errors.New("invalid or corrupted encrypted export")
	ErrInvalidRegistry = errors.New("profile registry is invalid or corrupted")
)

type profileMetadata struct {
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

type profileRegistry struct {
	Version  int                        `json:"version"`
	Profiles map[string]profileMetadata `json:"profiles"`
}

type ProfileService struct {
	storage port.KeyringStorage
	locker  port.ProcessLocker
	now     func() time.Time
}

func NewProfileService(storage port.KeyringStorage) *ProfileService {
	return &ProfileService{storage: storage, locker: port.NoopLocker{}, now: time.Now}
}

func NewProfileServiceWithLocker(storage port.KeyringStorage, locker port.ProcessLocker) *ProfileService {
	if locker == nil {
		locker = port.NoopLocker{}
	}
	return &ProfileService{storage: storage, locker: locker, now: time.Now}
}

// SetClock is intended for deterministic tests and does not affect production use.
func (s *ProfileService) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

func (s *ProfileService) CurrentProfile() (*domain.Profile, error) {
	unlock, err := s.acquireLock()
	if err != nil {
		return nil, err
	}
	defer unlock()

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

	name := "active"
	registry, registryErr := s.loadRegistry()
	if registryErr != nil {
		return nil, fmt.Errorf("failed to read profile registry: %w", registryErr)
	}
	for profileName, meta := range registry.Profiles {
		if saved, getErr := s.storage.Get(formatProfileKey(profileName)); getErr == nil && sameToken(saved, rawToken) {
			name = profileName
			return &domain.Profile{Name: name, Email: payload.IDTokenEmail, RawData: rawToken, CreatedAt: meta.CreatedAt, LastUsedAt: meta.LastUsedAt, Active: true}, nil
		}
	}
	return &domain.Profile{Name: name, Email: payload.IDTokenEmail, RawData: rawToken, Active: name != "active"}, nil
}

// SaveProfile preserves the original API and never overwrites an existing profile.
func (s *ProfileService) SaveProfile(name string) (*domain.Profile, error) {
	return s.saveProfile(name, false)
}

// SaveProfileForce explicitly replaces an existing profile.
func (s *ProfileService) SaveProfileForce(name string) (*domain.Profile, error) {
	return s.saveProfile(name, true)
}

func (s *ProfileService) saveProfile(name string, force bool) (*domain.Profile, error) {
	unlock, err := s.acquireLock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	if err := domain.ValidateProfileName(name); err != nil {
		return nil, err
	}
	rawToken, payload, err := s.activeToken()
	if err != nil {
		return nil, err
	}
	registry, err := s.loadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to read profile registry: %w", err)
	}
	if _, exists := registry.Profiles[name]; exists && !force {
		return nil, fmt.Errorf("%w: '%s'", ErrProfileExists, name)
	}
	previousRaw, previousErr := s.storage.Get(formatProfileKey(name))
	now := s.now().UTC()
	created := now
	if old, exists := registry.Profiles[name]; exists && !old.CreatedAt.IsZero() {
		created = old.CreatedAt
	}
	if err := s.storage.Set(formatProfileKey(name), rawToken); err != nil {
		return nil, fmt.Errorf("failed to save profile to keyring: %w", err)
	}
	registry.Profiles[name] = profileMetadata{CreatedAt: created, LastUsedAt: now}
	if err := s.saveRegistry(registry); err != nil {
		if previousErr == nil {
			_ = s.storage.Set(formatProfileKey(name), previousRaw)
		} else {
			_ = s.storage.Delete(formatProfileKey(name))
		}
		return nil, fmt.Errorf("failed to register profile name: %w", err)
	}
	return &domain.Profile{Name: name, Email: payload.IDTokenEmail, RawData: rawToken, CreatedAt: created, LastUsedAt: now}, nil
}

func (s *ProfileService) SwitchProfile(name string) (*domain.Profile, error) {
	unlock, err := s.acquireLock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	if err := domain.ValidateProfileName(name); err != nil {
		return nil, err
	}
	rawToken, err := s.storage.Get(formatProfileKey(name))
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
	registry, err := s.loadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to read profile registry: %w", err)
	}
	meta, exists := registry.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("%w: '%s'", ErrProfileNotFound, name)
	}
	if err := s.storage.Set(ActiveAccountKey, rawToken); err != nil {
		return nil, fmt.Errorf("failed to activate profile token: %w", err)
	}
	meta.LastUsedAt = s.now().UTC()
	registry.Profiles[name] = meta
	if err := s.saveRegistry(registry); err != nil {
		return nil, fmt.Errorf("profile activated but metadata update failed: %w", err)
	}
	return &domain.Profile{Name: name, Email: payload.IDTokenEmail, RawData: rawToken, CreatedAt: meta.CreatedAt, LastUsedAt: meta.LastUsedAt, Active: true}, nil
}

func (s *ProfileService) RenameProfile(oldName, newName string) error {
	unlock, err := s.acquireLock()
	if err != nil {
		return err
	}
	defer unlock()

	if err := domain.ValidateProfileName(oldName); err != nil {
		return err
	}
	if err := domain.ValidateProfileName(newName); err != nil {
		return err
	}
	registry, err := s.loadRegistry()
	if err != nil {
		return fmt.Errorf("failed to read profile registry: %w", err)
	}
	if _, ok := registry.Profiles[oldName]; !ok {
		return fmt.Errorf("%w: '%s'", ErrProfileNotFound, oldName)
	}
	if oldName == newName {
		return nil
	}
	if _, ok := registry.Profiles[newName]; ok {
		return fmt.Errorf("%w: '%s'", ErrProfileExists, newName)
	}
	if _, keyErr := s.storage.Get(formatProfileKey(newName)); keyErr == nil {
		return fmt.Errorf("%w: '%s'", ErrProfileExists, newName)
	}
	raw, err := s.storage.Get(formatProfileKey(oldName))
	if err != nil {
		return fmt.Errorf("failed to load profile token: %w", err)
	}
	if err := s.storage.Set(formatProfileKey(newName), raw); err != nil {
		return fmt.Errorf("failed to create renamed profile: %w", err)
	}
	meta := registry.Profiles[oldName]
	delete(registry.Profiles, oldName)
	registry.Profiles[newName] = meta
	if err := s.saveRegistry(registry); err != nil {
		_ = s.storage.Delete(formatProfileKey(newName))
		return fmt.Errorf("failed to update profile registry: %w", err)
	}
	if err := s.storage.Delete(formatProfileKey(oldName)); err != nil {
		// Restore registry and new key when the final move step fails.
		delete(registry.Profiles, newName)
		registry.Profiles[oldName] = meta
		_ = s.saveRegistry(registry)
		_ = s.storage.Delete(formatProfileKey(newName))
		return fmt.Errorf("failed to remove old profile: %w", err)
	}
	return nil
}

func (s *ProfileService) ListProfiles() ([]domain.Profile, error) {
	unlock, err := s.acquireLock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	registry, err := s.loadRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to read profile registry: %w", err)
	}
	active, _ := s.storage.Get(ActiveAccountKey)
	profiles := make([]domain.Profile, 0, len(registry.Profiles))
	for name, meta := range registry.Profiles {
		rawToken, getErr := s.storage.Get(formatProfileKey(name))
		if getErr != nil {
			continue
		}
		payload, parseErr := domain.ParseTokenPayload(rawToken)
		if parseErr != nil {
			continue
		}
		profiles = append(profiles, domain.Profile{
			Name: name, Email: payload.IDTokenEmail, RawData: rawToken,
			CreatedAt: meta.CreatedAt, LastUsedAt: meta.LastUsedAt, Active: active != "" && sameToken(active, rawToken),
		})
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	return profiles, nil
}

func (s *ProfileService) ProfileExists(name string) (bool, error) {
	unlock, err := s.acquireLock()
	if err != nil {
		return false, err
	}
	defer unlock()

	if err := domain.ValidateProfileName(name); err != nil {
		return false, err
	}
	registry, err := s.loadRegistry()
	if err != nil {
		return false, fmt.Errorf("failed to read profile registry: %w", err)
	}
	_, exists := registry.Profiles[name]
	return exists, nil
}

func (s *ProfileService) DeleteProfile(name string) error {
	unlock, err := s.acquireLock()
	if err != nil {
		return err
	}
	defer unlock()

	if err := domain.ValidateProfileName(name); err != nil {
		return err
	}
	registry, err := s.loadRegistry()
	if err != nil {
		return fmt.Errorf("failed to read profile registry: %w", err)
	}
	if _, exists := registry.Profiles[name]; !exists {
		return fmt.Errorf("%w: '%s'", ErrProfileNotFound, name)
	}
	if err := s.storage.Delete(formatProfileKey(name)); err != nil && !errors.Is(err, port.ErrNotFound) {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	delete(registry.Profiles, name)
	return s.saveRegistry(registry)
}

func (s *ProfileService) LogoutActive() error {
	unlock, err := s.acquireLock()
	if err != nil {
		return err
	}
	defer unlock()

	if err := s.storage.Delete(ActiveAccountKey); err != nil && !errors.Is(err, port.ErrNotFound) {
		return fmt.Errorf("failed to clear active session: %w", err)
	}
	return nil
}

func (s *ProfileService) CaptureActiveToken() (string, bool, error) {
	unlock, err := s.acquireLock()
	if err != nil {
		return "", false, err
	}
	defer unlock()

	raw, err := s.storage.Get(ActiveAccountKey)
	if errors.Is(err, port.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return raw, true, nil
}

func (s *ProfileService) RestoreActiveToken(raw string, existed bool) error {
	unlock, err := s.acquireLock()
	if err != nil {
		return err
	}
	defer unlock()

	if existed {
		return s.storage.Set(ActiveAccountKey, raw)
	}
	if err := s.storage.Delete(ActiveAccountKey); err != nil && !errors.Is(err, port.ErrNotFound) {
		return err
	}
	return nil
}

func (s *ProfileService) activeToken() (string, domain.TokenPayload, error) {
	raw, err := s.storage.Get(ActiveAccountKey)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return "", domain.TokenPayload{}, ErrNoActiveSession
		}
		return "", domain.TokenPayload{}, fmt.Errorf("failed to retrieve active token: %w", err)
	}
	payload, err := domain.ParseTokenPayload(raw)
	if err != nil {
		return "", domain.TokenPayload{}, fmt.Errorf("corrupted active token data: %w", err)
	}
	return raw, payload, nil
}

func (s *ProfileService) loadRegistry() (profileRegistry, error) {
	raw, err := s.storage.Get(RegistryAccountKey)
	if errors.Is(err, port.ErrNotFound) {
		return profileRegistry{Version: registryVersion, Profiles: map[string]profileMetadata{}}, nil
	}
	if err != nil {
		return profileRegistry{}, err
	}
	var registry profileRegistry
	if json.Unmarshal([]byte(raw), &registry) == nil && registry.Profiles != nil {
		if registry.Version == 0 {
			registry.Version = registryVersion
		}
		if registry.Version != registryVersion {
			return profileRegistry{}, ErrInvalidRegistry
		}
		for name := range registry.Profiles {
			if domain.ValidateProfileName(name) != nil {
				return profileRegistry{}, ErrInvalidRegistry
			}
		}
		return registry, nil
	}
	var legacy []string
	if json.Unmarshal([]byte(raw), &legacy) != nil {
		return profileRegistry{}, ErrInvalidRegistry
	}
	registry = profileRegistry{Version: registryVersion, Profiles: make(map[string]profileMetadata, len(legacy))}
	now := s.now().UTC()
	for _, name := range legacy {
		if domain.ValidateProfileName(name) == nil {
			registry.Profiles[name] = profileMetadata{CreatedAt: now, LastUsedAt: time.Time{}}
		}
	}
	// Persist migration lazily but immediately, so future writes use the new format.
	if err := s.saveRegistry(registry); err != nil {
		return profileRegistry{}, fmt.Errorf("failed to migrate profile registry: %w", err)
	}
	return registry, nil
}

func (s *ProfileService) saveRegistry(registry profileRegistry) error {
	registry.Version = registryVersion
	if registry.Profiles == nil {
		registry.Profiles = map[string]profileMetadata{}
	}
	data, err := json.Marshal(registry)
	if err != nil {
		return err
	}
	return s.storage.Set(RegistryAccountKey, string(data))
}

func formatProfileKey(profileName string) string { return ProfilePrefix + profileName }

func sameToken(left, right string) bool {
	if left == right {
		return true
	}
	var leftValue, rightValue any
	if json.Unmarshal([]byte(left), &leftValue) != nil || json.Unmarshal([]byte(right), &rightValue) != nil {
		return false
	}
	leftJSON, leftErr := json.Marshal(leftValue)
	rightJSON, rightErr := json.Marshal(rightValue)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func (s *ProfileService) acquireLock() (func(), error) {
	if s.locker == nil {
		return func() {}, nil
	}
	if err := s.locker.Lock(); err != nil {
		return nil, fmt.Errorf("failed to acquire profile lock: %w", err)
	}
	return func() {
		_ = s.locker.Unlock()
	}, nil
}
