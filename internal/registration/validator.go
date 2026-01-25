package registration

import (
	"errors"
	"slices"

	"github.com/baighasan/kubecraft/internal/config"
)

func ValidateUsername(username string) error {
	// Check length
	if len(username) < 3 || len(username) > 16 {
		return errors.New("username must be 3-16 characters long")
	}

	// Check format (lowercase a-z and numbers 0-9)
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return errors.New("username must only contain lowercase letters and numbers")
		}
	}

	// Check username starts with a letter
	if username[0] < 'a' || username[0] > 'z' {
		return errors.New("username must start with a lowercase letter")
	}

	// Check against reserved names
	if slices.Contains(config.ReservedUserNames, username) {
		return errors.New("username is reserved")
	}

	return nil
}
