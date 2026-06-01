package server

import (
	"os"
	"testing"

	"github.com/baighasan/kubecraft/internal/config"
)

func TestCreateCmdArgsPolicy(t *testing.T) {
	if err := createCmd.Args(createCmd, []string{}); err != nil {
		t.Fatalf("createCmd.Args with zero args returned error: %v", err)
	}

	if err := createCmd.Args(createCmd, []string{"myserver"}); err != nil {
		t.Fatalf("createCmd.Args with one arg returned error: %v", err)
	}

	if err := createCmd.Args(createCmd, []string{"a", "b"}); err == nil {
		t.Fatal("createCmd.Args with two args expected error, got nil")
	}
}

func TestBuildDefaultCreateInput(t *testing.T) {
	serverName := "myserver"
	input := buildDefaultCreateInput(serverName)

	if input.ServerName != serverName {
		t.Errorf("ServerName = %q, want %q", input.ServerName, serverName)
	}

	if input.Version != config.DefaultMinecraftVersion {
		t.Errorf("Version = %q, want %q", input.Version, config.DefaultMinecraftVersion)
	}

	if input.GameMode != config.DefaultGameMode {
		t.Errorf("GameMode = %q, want %q", input.GameMode, config.DefaultGameMode)
	}

	if input.MaxPlayers != config.DefaultMaxPlayers {
		t.Errorf("MaxPlayers = %d, want %d", input.MaxPlayers, config.DefaultMaxPlayers)
	}
}

func TestResolveServerImage(t *testing.T) {
	t.Setenv(serverImageEnvVar, "env-image")

	resolved := resolveServerImage("flag-image")
	if resolved != "flag-image" {
		t.Errorf("resolveServerImage(flag) = %q, want %q", resolved, "flag-image")
	}

	resolved = resolveServerImage("")
	if resolved != "env-image" {
		t.Errorf("resolveServerImage(env) = %q, want %q", resolved, "env-image")
	}

	if err := os.Unsetenv(serverImageEnvVar); err != nil {
		t.Fatalf("failed to unset env var: %v", err)
	}

	resolved = resolveServerImage("")
	if resolved != "" {
		t.Errorf("resolveServerImage(default) = %q, want empty string", resolved)
	}
}

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
