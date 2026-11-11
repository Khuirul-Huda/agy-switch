package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/khuirulhuda/agy-switch/internal/domain"
	"github.com/khuirulhuda/agy-switch/internal/port"
)

const (
	exportMagic   = "AGY-SWITCH-EXPORT"
	exportVersion = 1
	kdfIterations = 120000
)

type encryptedExport struct {
	Magic      string `json:"magic"`
	Version    int    `json:"version"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type exportPayload struct {
	Version  int                        `json:"version"`
	Profiles map[string]exportedProfile `json:"profiles"`
}

type exportedProfile struct {
	Email      string `json:"email"`
	RawData    string `json:"token"`
	CreatedAt  string `json:"created_at,omitempty"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

func (s *ProfileService) ExportProfiles(fs port.FileSystem, path, password string) error {
	if password == "" {
		return errors.New("export password cannot be empty")
	}
	profiles, err := s.ListProfiles()
	if err != nil {
		return err
	}
	payload := exportPayload{Version: exportVersion, Profiles: make(map[string]exportedProfile, len(profiles))}
	for _, profile := range profiles {
		payload.Profiles[profile.Name] = exportedProfile{
			Email: profile.Email, RawData: profile.RawData,
			CreatedAt:  profile.CreatedAt.UTC().Format(timeFormat),
			LastUsedAt: profile.LastUsedAt.UTC().Format(timeFormat),
		}
	}
	plain, err := json.Marshal(payload)
	if err != nil {
		return errors.New("failed to prepare export")
	}
	blob, err := encryptExport(plain, password)
	if err != nil {
		return errors.New("failed to encrypt export")
	}
	tempPath, err := temporaryExportPath(path)
	if err != nil {
		return errors.New("failed to prepare export file")
	}
	if err := fs.WriteFile(tempPath, blob, 0600); err != nil {
		_ = fs.Remove(tempPath)
		return fmt.Errorf("failed to write export file: %w", err)
	}
	if err := fs.Chmod(tempPath, 0600); err != nil {
		_ = fs.Remove(tempPath)
		return fmt.Errorf("failed to secure export file: %w", err)
	}
	if err := fs.Rename(tempPath, path); err != nil {
		_ = fs.Remove(tempPath)
		return fmt.Errorf("failed to replace export file: %w", err)
	}
	if err := fs.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to secure export file: %w", err)
	}
	return nil
}

func (s *ProfileService) ImportProfiles(fs port.FileSystem, path, password string, force bool) error {
	unlock, err := s.acquireLock()
	if err != nil {
		return err
	}
	defer unlock()

	if password == "" {
		return errors.New("import password cannot be empty")
	}
	blob, err := fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read export file: %w", err)
	}
	plain, err := decryptExport(blob, password)
	if err != nil {
		return ErrInvalidExport
	}
	var payload exportPayload
	if json.Unmarshal(plain, &payload) != nil || payload.Version != exportVersion || payload.Profiles == nil {
		return ErrInvalidExport
	}
	registry, err := s.loadRegistry()
	if err != nil {
		return fmt.Errorf("failed to read profile registry: %w", err)
	}
	metadata := make(map[string]profileMetadata, len(payload.Profiles))
	previous := make(map[string]struct {
		raw    string
		exists bool
		meta   profileMetadata
	}, len(payload.Profiles))
	for name, exported := range payload.Profiles {
		if err := domain.ValidateProfileName(name); err != nil || exported.RawData == "" {
			return ErrInvalidExport
		}
		if _, exists := registry.Profiles[name]; exists && !force {
			return fmt.Errorf("%w: '%s'", ErrProfileExists, name)
		}
		previousRaw, previousErr := s.storage.Get(formatProfileKey(name))
		previous[name] = struct {
			raw    string
			exists bool
			meta   profileMetadata
		}{raw: previousRaw, exists: previousErr == nil, meta: registry.Profiles[name]}
		if _, err := domain.ParseTokenPayload(exported.RawData); err != nil {
			return ErrInvalidExport
		}
		meta := profileMetadata{}
		if exported.CreatedAt != "" {
			meta.CreatedAt, err = parseTime(exported.CreatedAt)
			if err != nil {
				return ErrInvalidExport
			}
		}
		if exported.LastUsedAt != "" {
			meta.LastUsedAt, err = parseTime(exported.LastUsedAt)
			if err != nil {
				return ErrInvalidExport
			}
		}
		if meta.CreatedAt.IsZero() {
			meta.CreatedAt = s.now().UTC()
		}
		metadata[name] = meta
	}
	written := make([]string, 0, len(payload.Profiles))
	for name, exported := range payload.Profiles {
		if err := s.storage.Set(formatProfileKey(name), exported.RawData); err != nil {
			for _, writtenName := range written {
				old := previous[writtenName]
				if old.exists {
					_ = s.storage.Set(formatProfileKey(writtenName), old.raw)
				} else {
					_ = s.storage.Delete(formatProfileKey(writtenName))
				}
			}
			return errors.New("failed to import profile into keyring")
		}
		written = append(written, name)
		registry.Profiles[name] = metadata[name]
	}
	if err := s.saveRegistry(registry); err != nil {
		for _, name := range written {
			old := previous[name]
			if old.exists {
				_ = s.storage.Set(formatProfileKey(name), old.raw)
			} else {
				_ = s.storage.Delete(formatProfileKey(name))
			}
		}
		_ = s.saveRegistry(profileRegistry{Version: registry.Version, Profiles: cloneMetadata(registry, previous)})
		return errors.New("failed to update profile registry")
	}
	return nil
}

func temporaryExportPath(path string) (string, error) {
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.tmp-%x", path, suffix), nil
}

func cloneMetadata(registry profileRegistry, previous map[string]struct {
	raw    string
	exists bool
	meta   profileMetadata
}) map[string]profileMetadata {
	result := make(map[string]profileMetadata, len(registry.Profiles))
	for name, meta := range registry.Profiles {
		result[name] = meta
	}
	for name, old := range previous {
		if old.exists {
			result[name] = old.meta
		} else {
			delete(result, name)
		}
	}
	return result
}

func encryptExport(plain []byte, password string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	key := deriveKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plain, []byte(exportMagic))
	out, err := json.Marshal(encryptedExport{
		Magic: exportMagic, Version: exportVersion,
		Salt:       base64.RawStdEncoding.EncodeToString(salt),
		Nonce:      base64.RawStdEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawStdEncoding.EncodeToString(ciphertext),
	})
	return out, err
}

func decryptExport(blob []byte, password string) ([]byte, error) {
	var envelope encryptedExport
	if json.Unmarshal(blob, &envelope) != nil || envelope.Magic != exportMagic || envelope.Version != exportVersion {
		return nil, ErrInvalidExport
	}
	salt, err := base64.RawStdEncoding.DecodeString(envelope.Salt)
	if err != nil || len(salt) < 16 {
		return nil, ErrInvalidExport
	}
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, ErrInvalidExport
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, ErrInvalidExport
	}
	block, err := aes.NewCipher(deriveKey(password, salt))
	if err != nil {
		return nil, ErrInvalidExport
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrInvalidExport
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, []byte(exportMagic))
	if err != nil {
		return nil, ErrInvalidExport
	}
	return plain, nil
}

func deriveKey(password string, salt []byte) []byte {
	// PBKDF2-HMAC-SHA256 implemented with the standard library only.
	prf := hmac.New(sha256.New, []byte(password))
	prf.Write(salt)
	prf.Write([]byte{0, 0, 0, 1})
	u := prf.Sum(nil)
	result := append([]byte(nil), u...)
	for i := 1; i < kdfIterations; i++ {
		prf = hmac.New(sha256.New, []byte(password))
		prf.Write(u)
		u = prf.Sum(nil)
		for j := range result {
			result[j] ^= u[j]
		}
	}
	return result
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

func parseTime(value string) (time.Time, error) {
	return time.Parse(timeFormat, value)
}

var _ port.FileSystem = port.OSFileSystem{}
