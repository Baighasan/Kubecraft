package server

import (
	"testing"
)

func TestValidateServerName_Valid(t *testing.T) {
	validNames := []string{
		"abc",
		"myserver",
		"server1",
		"a1b2c3",
		"abcdefghijklmnop", // 16 chars (max)
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			if err := ValidateServerName(name); err != nil {
				t.Errorf("ValidateServerName(%q) error = %v, want nil", name, err)
			}
		})
	}
}

func TestValidateServerName_TooShort(t *testing.T) {
	shortNames := []string{"", "a", "ab"}

	for _, name := range shortNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateServerName(name)
			if err == nil {
				t.Errorf("ValidateServerName(%q) expected error, got nil", name)
			}
		})
	}
}

func TestValidateServerName_TooLong(t *testing.T) {
	err := ValidateServerName("abcdefghijklmnopq") // 17 chars
	if err == nil {
		t.Error("ValidateServerName() expected error for 17-char name, got nil")
	}
}

func TestValidateServerName_UppercaseRejected(t *testing.T) {
	invalidNames := []string{"MyServer", "ALLCAPS", "serverA"}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateServerName(name)
			if err == nil {
				t.Errorf("ValidateServerName(%q) expected error, got nil", name)
			}
		})
	}
}

func TestValidateServerName_SpecialCharsRejected(t *testing.T) {
	invalidNames := []string{"my-server", "my_server", "my.server", "my server", "server!"}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateServerName(name)
			if err == nil {
				t.Errorf("ValidateServerName(%q) expected error, got nil", name)
			}
		})
	}
}

func TestValidateServerName_MustStartWithLetter(t *testing.T) {
	invalidNames := []string{"1server", "123", "9abc"}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateServerName(name)
			if err == nil {
				t.Errorf("ValidateServerName(%q) expected error, got nil", name)
			}
		})
	}
}
