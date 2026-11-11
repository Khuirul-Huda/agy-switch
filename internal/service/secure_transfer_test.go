package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/khuirulhuda/agy-switch/internal/adapter"
	"github.com/khuirulhuda/agy-switch/internal/port"
	"github.com/khuirulhuda/agy-switch/internal/service"
)

type testFS struct{ root string }

func testDirectory(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".agy-switch-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func (f testFS) path(name string) string              { return filepath.Join(f.root, filepath.Base(name)) }
func (f testFS) ReadFile(name string) ([]byte, error) { return os.ReadFile(f.path(name)) }
func (f testFS) WriteFile(name string, data []byte, mode os.FileMode) error {
	return os.WriteFile(f.path(name), data, mode)
}
func (f testFS) Chmod(name string, mode os.FileMode) error { return os.Chmod(f.path(name), mode) }
func (f testFS) Rename(oldName, newName string) error {
	return os.Rename(f.path(oldName), f.path(newName))
}
func (f testFS) Remove(name string) error              { return os.Remove(f.path(name)) }
func (f testFS) Stat(name string) (os.FileInfo, error) { return os.Stat(f.path(name)) }

var _ port.FileSystem = testFS{}

func TestEncryptedExportImportRoundTrip(t *testing.T) {
	dir := testDirectory(t)
	fs := testFS{root: dir}
	sourceStorage := adapter.NewMemoryKeyring()
	source := service.NewProfileService(sourceStorage)
	_ = sourceStorage.Set(service.ActiveAccountKey, sampleRawToken)
	if _, err := source.SaveProfile("personal"); err != nil {
		t.Fatal(err)
	}
	if err := source.ExportProfiles(fs, "profiles.agy", "correct horse"); err != nil {
		t.Fatal(err)
	}
	blob, err := os.ReadFile(filepath.Join(dir, "profiles.agy"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), "access_token") || strings.Contains(string(blob), "abc123xyz") {
		t.Fatal("export contains plaintext token data")
	}
	info, err := os.Stat(filepath.Join(dir, "profiles.agy"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected export mode 0600, got %o", info.Mode().Perm())
	}

	targetStorage := adapter.NewMemoryKeyring()
	target := service.NewProfileService(targetStorage)
	if err := target.ImportProfiles(fs, "profiles.agy", "correct horse", false); err != nil {
		t.Fatal(err)
	}
	profile, err := target.SwitchProfile("personal")
	if err != nil || profile.Email != "test@example.com" {
		t.Fatalf("round-trip profile unavailable: %v", err)
	}
}

func TestEncryptedExportRejectsWrongPasswordAndMalformedInput(t *testing.T) {
	dir := testDirectory(t)
	fs := testFS{root: dir}
	storage := adapter.NewMemoryKeyring()
	svc := service.NewProfileService(storage)
	_ = storage.Set(service.ActiveAccountKey, sampleRawToken)
	_, _ = svc.SaveProfile("personal")
	if err := svc.ExportProfiles(fs, "profiles.agy", "secret"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ImportProfiles(fs, "profiles.agy", "wrong", false); err != service.ErrInvalidExport {
		t.Fatalf("expected wrong password error, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.agy"), []byte("not an export"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := svc.ImportProfiles(fs, "bad.agy", "secret", false); err != service.ErrInvalidExport {
		t.Fatalf("expected malformed export error, got %v", err)
	}
}
