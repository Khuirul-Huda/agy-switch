package domain_test

import (
	"errors"
	"testing"

	"github.com/khuirulhuda/agy-switch/internal/domain"
)

func TestValidateProfileName_ValidNames(t *testing.T) {
	// Arrange
	validNames := []string{
		"personal",
		"work",
		"client-1",
		"client_2",
		"Account123",
		"a",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			// Act
			err := domain.ValidateProfileName(name)

			// Assert
			if err != nil {
				t.Fatalf("expected valid name for '%s', got error: %v", name, err)
			}
		})
	}
}

func TestValidateProfileName_InvalidNames(t *testing.T) {
	// Arrange
	invalidNames := []string{
		"",
		"   ",
		"name with spaces",
		"name/with/slash",
		"name:with:colon",
		"this-name-is-definitely-way-too-long-to-be-considered-valid-in-our-system",
		"bad@email",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			// Act
			err := domain.ValidateProfileName(name)

			// Assert
			if !errors.Is(err, domain.ErrInvalidProfileName) {
				t.Fatalf("expected ErrInvalidProfileName for '%s', got: %v", name, err)
			}
		})
	}
}

func TestParseTokenPayload_Success(t *testing.T) {
	// Arrange
	rawJSON := `{"auth_method":"consumer","id_token_email":"user@gmail.com","token":{"access_token":"abc"}}`

	// Act
	payload, err := domain.ParseTokenPayload(rawJSON)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error parsing payload: %v", err)
	}
	if payload.IDTokenEmail != "user@gmail.com" {
		t.Errorf("expected email 'user@gmail.com', got '%s'", payload.IDTokenEmail)
	}
	if payload.AuthMethod != "consumer" {
		t.Errorf("expected auth_method 'consumer', got '%s'", payload.AuthMethod)
	}
}

func TestParseTokenPayload_EmptyPayload(t *testing.T) {
	// Arrange
	emptyInputs := []string{"", "   ", "\n"}

	for _, raw := range emptyInputs {
		t.Run("empty", func(t *testing.T) {
			// Act
			_, err := domain.ParseTokenPayload(raw)

			// Assert
			if !errors.Is(err, domain.ErrEmptyTokenPayload) {
				t.Fatalf("expected ErrEmptyTokenPayload, got: %v", err)
			}
		})
	}
}

func TestParseTokenPayload_InvalidJSON(t *testing.T) {
	// Arrange
	malformedJSON := `{"auth_method": incomplete json`

	// Act
	_, err := domain.ParseTokenPayload(malformedJSON)

	// Assert
	if !errors.Is(err, domain.ErrInvalidJSONPayload) {
		t.Fatalf("expected ErrInvalidJSONPayload, got: %v", err)
	}
}
