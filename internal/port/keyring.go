package port

import "errors"

var (
	// ErrNotFound indicates that the requested key/account does not exist in the keyring.
	ErrNotFound = errors.New("specified keyring account not found")
)

// KeyringStorage defines the minimal, cross-platform contract for credential storage operations.
type KeyringStorage interface {
	// Get retrieves a secret by account name.
	Get(account string) (string, error)

	// Set stores a secret under the specified account name.
	Set(account string, secret string) error

	// Delete removes a secret by account name.
	Delete(account string) error
}
