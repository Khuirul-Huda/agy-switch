package port

import (
	"os"
)

// FileSystem is the minimal file contract used by encrypted export/import.
// Keeping it as a port makes those operations hermetic in tests.
type FileSystem interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Chmod(name string, mode os.FileMode) error
	Rename(oldPath, newPath string) error
	Remove(name string) error
}

// OSFileSystem is the production filesystem adapter.
type OSFileSystem struct{}

func (OSFileSystem) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (OSFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (OSFileSystem) Chmod(name string, mode os.FileMode) error { return os.Chmod(name, mode) }
func (OSFileSystem) Rename(oldPath, newPath string) error      { return os.Rename(oldPath, newPath) }
func (OSFileSystem) Remove(name string) error                  { return os.Remove(name) }
