package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// createFakeConfig creates a config file in the test HOME directory
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

func TestRegisterUser_AlreadyRegistered(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	createFakeConfig(t)

	err := registerUser("newuser")
	if err == nil {
		t.Fatal("expected error when already registered, got nil")
	}

	expected := "you are already registered. Delete ~/.kubecraft/config first if you want to re-register"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestRegisterUserAtURL_Unreachable(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	// Start and immediately close a server to get a port that refuses connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL + "/register"
	server.Close()

	err := registerUserAtURL("alice", url)
	if err == nil {
		t.Fatal("expected error when server unreachable, got nil")
	}

	expected := "could not reach registration server at"
	if len(err.Error()) < len(expected) || err.Error()[:len(expected)] != expected {
		t.Errorf("error = %q, want prefix %q", err.Error(), expected)
	}
}

func TestConfig_NoClusterEndpoint(t *testing.T) {
	cfg := &config.Config{
		Username: "testuser",
		Token:    "testtoken",
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestConfig_SaveAndLoad_NoClusterEndpoint(t *testing.T) {
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
}

func TestClusterEndpoint_Default(t *testing.T) {
	if config.ClusterEndpoint == "" {
		t.Error("ClusterEndpoint should have a default value")
	}
}

// Tests below use registerUserAtURL to test HTTP interaction logic
// without being constrained by the const port in registerUser.

func TestRegisterUserAtURL_Success(t *testing.T) {
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

	err := registerUserAtURL("alice", server.URL+"/register")
	if err != nil {
		t.Fatalf("registerUserAtURL() error = %v", err)
	}

	// Verify config was saved
	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.Username != "alice" {
		t.Errorf("saved Username = %q, want %q", loaded.Username, "alice")
	}
	if loaded.Token != "test-token-abc123" {
		t.Errorf("saved Token = %q, want %q", loaded.Token, "test-token-abc123")
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

	err := registerUserAtURL("alice", server.URL+"/register")
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

	err := registerUserAtURL("alice", server.URL+"/register")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expected := "registration server returned status 502 and response could not be parsed"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

func TestRegisterUserAtURL_AlreadyRegistered(t *testing.T) {
	cleanup := setTestHome(t)
	defer cleanup()

	createFakeConfig(t)

	err := registerUserAtURL("newuser", "http://localhost/register")
	if err == nil {
		t.Fatal("expected error when already registered, got nil")
	}

	expected := "you are already registered. Delete ~/.kubecraft/config first if you want to re-register"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}
