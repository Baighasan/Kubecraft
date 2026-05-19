package cli

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baighasan/kubecraft/internal/config"
)

// setTestHome overrides HOME to a temp directory so config operations
// don't touch the real filesystem. Returns a cleanup function.
func setTestHome(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	return func() {
		os.Setenv("HOME", origHome)
	}
}

// createFakeConfig creates a config file in the test HOME directory with full credentials
func createFakeConfig(t *testing.T) {
	t.Helper()

	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	err = os.WriteFile(configPath, []byte("username: existinguser\ntoken: fake-token\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to write fake config: %v", err)
	}
}

// createInitOnlyConfig creates a config file with only cluster IP set (no credentials)
func createInitOnlyConfig(t *testing.T, clusterIP string) {
	t.Helper()

	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	content := "clusterIP: " + clusterIP + "\n"
	err = os.WriteFile(configPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to write init config: %v", err)
	}
}

func TestRegisterUser_ConfigMissing_ReturnsInitHint(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	err := registerUser("alice")
	if err == nil {
		t.Fatal("expected error when config missing, got nil")
	}

	expected := "config does not exist. run kubecraft init --ip <clusterIP> first"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestRegisterUser_BlocksWhenAlreadyRegistered(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	createFakeConfig(t)

	err := registerUser("newuser")
	if err == nil {
		t.Fatal("expected error when already registered, got nil")
	}

	expected := "you are already registered. delete user and token fields in ~/.kubecraft/config to register again"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestConfig_ValidateForServer_MissingClusterIP(t *testing.T) {
	cfg := &config.Config{
		Username: "testuser",
		Token:    "testtoken",
	}

	err := cfg.ValidateForServer()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, config.ErrClusterNotInitialized) {
		t.Fatalf("expected ErrClusterNotInitialized, got %v", err)
	}
}

func TestConfig_SaveAndLoad_DoesNotAutoValidate(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	cfg := &config.Config{
		Username: "alice",
		Token:    "my-token",
	}

	err := config.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Username != "alice" {
		t.Errorf("Username = %q, want %q", loaded.Username, "alice")
	}
	if loaded.Token != "my-token" {
		t.Errorf("Token = %q, want %q", loaded.Token, "my-token")
	}

	// LoadConfig should not auto-validate; explicit validation required
	if err := loaded.ValidateForRegister(); !errors.Is(err, config.ErrClusterNotInitialized) {
		t.Fatalf("ValidateForRegister expected ErrClusterNotInitialized, got %v", err)
	}
	if err := loaded.ValidateForServer(); !errors.Is(err, config.ErrClusterNotInitialized) {
		t.Fatalf("ValidateForServer expected ErrClusterNotInitialized, got %v", err)
	}
}

// Tests below use registerUserAtURL to test HTTP interaction logic
// without being constrained by the const port in registerUser.

func TestRegisterUserAtURL_Success_ReturnsCredentials(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/register" {
			t.Errorf("expected /register path, got %s", r.URL.Path)
		}

		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Username != "alice" {
			t.Errorf("username = %q, want %q", req.Username, "alice")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RegisterResponse{
			Status:   "success",
			Username: "alice",
			Token:    "test-token-abc123",
		})
	}))
	defer server.Close()

	creds, err := registerUserAtURL("alice", server.URL)
	if err != nil {
		t.Fatalf("registerUserAtURL() error = %v", err)
	}

	if creds == nil {
		t.Fatal("expected non-nil credentials, got nil")
	}
	if creds.Username != "alice" {
		t.Errorf("Username = %q, want %q", creds.Username, "alice")
	}
	if creds.Token != "test-token-abc123" {
		t.Errorf("Token = %q, want %q", creds.Token, "test-token-abc123")
	}
}

func TestRegisterUserAtURL_ServerReturnsError(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(RegisterResponse{
			Status:  "error",
			Message: "Username already registered",
		})
	}))
	defer server.Close()

	_, err := registerUserAtURL("alice", server.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expected := "failed to register user: Username already registered"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestRegisterUserAtURL_UnparseableResponse(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("<html>Bad Gateway</html>"))
	}))
	defer server.Close()

	_, err := registerUserAtURL("alice", server.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expected := "registration server returned status 502 and response could not be parsed"
	if !strings.HasPrefix(err.Error(), expected) {
		t.Errorf("error = %q, want prefix %q", err.Error(), expected)
	}
}

func TestRegisterUserAtURL_Unreachable(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	// Start and immediately close a server to get a port that refuses connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	_, err := registerUserAtURL("alice", url)
	if err == nil {
		t.Fatal("expected error when server unreachable, got nil")
	}

	expected := "could not reach registration server at"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("error = %q, want prefix %q", err.Error(), expected)
	}
}
