package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrInvalidProfileName indicates that the requested profile name violates naming constraints.
	ErrInvalidProfileName = errors.New("profile name must be 1-32 alphanumeric characters and may include hyphens or underscores")

	// ErrEmptyTokenPayload indicates an empty secret payload.
	ErrEmptyTokenPayload = errors.New("token payload cannot be empty")

	// ErrInvalidJSONPayload indicates corrupted or non-JSON token data.
	ErrInvalidJSONPayload = errors.New("invalid token JSON structure")

	profileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
)

// TokenPayload models the structure stored by agy inside the OS keyring.
type TokenPayload struct {
	AuthMethod   string `json:"auth_method"`
	IDTokenEmail string `json:"id_token_email"`
	Token        any    `json:"token"`
}

// Profile represents a named account configuration.
type Profile struct {
	Name    string
	Email   string
	RawData string
}

// ValidateProfileName validates that a profile name conforms to safe naming standards.
func ValidateProfileName(name string) error {
	trimmed := strings.TrimSpace(name)
	if !profileNamePattern.MatchString(trimmed) {
		return fmt.Errorf("%w: '%s'", ErrInvalidProfileName, name)
	}
	return nil
}

// ParseTokenPayload decodes the raw JSON token string into a structured TokenPayload.
func ParseTokenPayload(rawJSON string) (TokenPayload, error) {
	if strings.TrimSpace(rawJSON) == "" {
		return TokenPayload{}, ErrEmptyTokenPayload
	}

	var payload TokenPayload
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return TokenPayload{}, fmt.Errorf("%w: %v", ErrInvalidJSONPayload, err)
	}

	if payload.IDTokenEmail == "" {
		payload.IDTokenEmail = "unknown"
	}

	return payload, nil
}
