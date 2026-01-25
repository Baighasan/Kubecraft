package registration

import (
	"testing"
)

func TestValidateUsername_Success(t *testing.T) {
	validUsernames := []string{
		"alice",
		"bob123",
		"user1",
		"test",
		"abc",              // minimum length (3)
		"a123456789012345", // maximum length (16)
	}

	for _, username := range validUsernames {
		t.Run(username, func(t *testing.T) {
			err := ValidateUsername(username)
			if err != nil {
				t.Errorf("ValidateUsername(%q) unexpected error: %v", username, err)
			}
		})
	}
}

func TestValidateUsername_TooShort(t *testing.T) {
	invalidUsernames := []string{
		"a",  // 1 char
		"ab", // 2 chars
		"",   // empty
	}

	expectedError := "username must be 3-16 characters long"

	for _, username := range invalidUsernames {
		t.Run(username, func(t *testing.T) {
			err := ValidateUsername(username)
			if err == nil {
				t.Errorf("ValidateUsername(%q) expected error, got nil", username)
			}
			if err.Error() != expectedError {
				t.Errorf("ValidateUsername(%q) error = %q, want %q", username, err.Error(), expectedError)
			}
		})
	}
}

func TestValidateUsername_TooLong(t *testing.T) {
	username := "a12345678901234567" // 18 chars
	expectedError := "username must be 3-16 characters long"

	err := ValidateUsername(username)
	if err == nil {
		t.Errorf("ValidateUsername(%q) expected error, got nil", username)
	}
	if err.Error() != expectedError {
		t.Errorf("ValidateUsername(%q) error = %q, want %q", username, err.Error(), expectedError)
	}
}

func TestValidateUsername_InvalidCharacters(t *testing.T) {
	tests := []struct {
		username string
		reason   string
	}{
		{"alice_bob", "underscore"},
		{"alice-bob", "hyphen"},
		{"alice.bob", "period"},
		{"alice bob", "space"},
		{"Alice", "uppercase letter"},
		{"alice!", "special character"},
		{"alice@", "at symbol"},
		{"alice#", "hash"},
		{"alice$", "dollar sign"},
	}

	expectedError := "username must only contain lowercase letters and numbers"

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if err == nil {
				t.Errorf("ValidateUsername(%q) expected error for %s, got nil", tt.username, tt.reason)
			}
			if err.Error() != expectedError {
				t.Errorf("ValidateUsername(%q) error = %q, want %q", tt.username, err.Error(), expectedError)
			}
		})
	}
}

func TestValidateUsername_MustStartWithLetter(t *testing.T) {
	tests := []string{
		"1alice", // starts with number
		"2test",  // starts with number
		"9user",  // starts with number
		"123abc", // starts with number
	}

	expectedError := "username must start with a lowercase letter"

	for _, username := range tests {
		t.Run(username, func(t *testing.T) {
			err := ValidateUsername(username)
			if err == nil {
				t.Errorf("ValidateUsername(%q) expected error, got nil", username)
			}
			if err.Error() != expectedError {
				t.Errorf("ValidateUsername(%q) error = %q, want %q", username, err.Error(), expectedError)
			}
		})
	}
}

func TestValidateUsername_ReservedNames(t *testing.T) {
	// Test all reserved names from config
	reservedNames := []string{
		"system",
		"admin",
		"root",
		"default",
		"kubecraft",
	}

	expectedError := "username is reserved"

	for _, username := range reservedNames {
		t.Run(username, func(t *testing.T) {
			err := ValidateUsername(username)
			if err == nil {
				t.Errorf("ValidateUsername(%q) expected error for reserved name, got nil", username)
			}
			if err.Error() != expectedError {
				t.Errorf("ValidateUsername(%q) error = %q, want %q", username, err.Error(), expectedError)
			}
		})
	}
}

func TestValidateUsername_EdgeCases(t *testing.T) {
	tests := []struct {
		username      string
		shouldSucceed bool
		errorMessage  string
	}{
		{"abc", true, ""},                                      // exactly 3 chars
		{"a123456789012345", true, ""},                         // exactly 16 chars
		{"test123", true, ""},                                  // mixed letters and numbers
		{"ab", false, "username must be 3-16 characters long"}, // 2 chars
		{"a12345678901234567", false, "username must be 3-16 characters long"}, // 17 chars
	}

	for _, tt := range tests {
		t.Run(tt.username, func(t *testing.T) {
			err := ValidateUsername(tt.username)

			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("ValidateUsername(%q) unexpected error: %v", tt.username, err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateUsername(%q) expected error, got nil", tt.username)
				} else if err.Error() != tt.errorMessage {
					t.Errorf("ValidateUsername(%q) error = %q, want %q", tt.username, err.Error(), tt.errorMessage)
				}
			}
		})
	}
}
