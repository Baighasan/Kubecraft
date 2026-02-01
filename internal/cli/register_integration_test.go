//go:build integration

package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/baighasan/kubecraft/internal/config"
	"github.com/baighasan/kubecraft/internal/k8s"
	"github.com/baighasan/kubecraft/internal/registration"
	"k8s.io/client-go/tools/clientcmd"
)

// TestRegisterIntegration_EndToEnd tests the full registration flow:
// CLI register function -> mock HTTP server with real registration handler -> real K8s cluster
func TestRegisterIntegration_EndToEnd(t *testing.T) {
	// Use a temp HOME so we don't touch real config
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(origHome, ".kube", "config")
	}
	os.Setenv("KUBECONFIG", kubeconfig)

	client := getIntegrationTestClient(t)
	ensureTestSystemRBAC(t, client)
	handler := registration.NewRegistrationHandler(client)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}))
	defer server.Close()

	username := uniqueTestUsername()
	defer cleanupTestNamespace(t, client, username)
	defer cleanupTestClusterRoleBinding(t, client, username)

	err := registerUserAtURL(username, server.URL+"/register")
	if err != nil {
		t.Fatalf("registerUserAtURL() error = %v", err)
	}

	// Verify config was saved correctly
	loaded, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Username != username {
		t.Errorf("saved Username = %q, want %q", loaded.Username, username)
	}

	if loaded.Token == "" {
		t.Error("saved Token is empty, expected a valid token")
	}

	// Verify namespace was created in K8s
	exists, err := client.NamespaceExists(username)
	if err != nil {
		t.Fatalf("NamespaceExists() error = %v", err)
	}
	if !exists {
		t.Errorf("namespace for user %q was not created", username)
	}
}

// TestRegisterIntegration_DuplicateBlockedByConfig tests that registering
// a second time is blocked by the existing config file
func TestRegisterIntegration_DuplicateBlockedByConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(origHome, ".kube", "config")
	}
	os.Setenv("KUBECONFIG", kubeconfig)

	client := getIntegrationTestClient(t)
	ensureTestSystemRBAC(t, client)
	handler := registration.NewRegistrationHandler(client)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}))
	defer server.Close()

	username := uniqueTestUsername()
	defer cleanupTestNamespace(t, client, username)
	defer cleanupTestClusterRoleBinding(t, client, username)

	// First registration should succeed
	err := registerUserAtURL(username, server.URL+"/register")
	if err != nil {
		t.Fatalf("first registration error = %v", err)
	}

	// Second registration should be blocked by existing config
	err = registerUserAtURL("otheruser", server.URL+"/register")
	if err == nil {
		t.Fatal("expected error on second registration, got nil")
	}

	expected := "you are already registered. Delete ~/.kubecraft/config first if you want to re-register"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}

// TestRegisterIntegration_ServerRejectsDuplicate tests that the server
// rejects a duplicate username even if the config doesn't exist locally
func TestRegisterIntegration_ServerRejectsDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(origHome, ".kube", "config")
	}
	os.Setenv("KUBECONFIG", kubeconfig)

	client := getIntegrationTestClient(t)
	ensureTestSystemRBAC(t, client)
	handler := registration.NewRegistrationHandler(client)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}))
	defer server.Close()

	username := uniqueTestUsername()
	defer cleanupTestNamespace(t, client, username)
	defer cleanupTestClusterRoleBinding(t, client, username)

	// First registration
	err := registerUserAtURL(username, server.URL+"/register")
	if err != nil {
		t.Fatalf("first registration error = %v", err)
	}

	// Delete local config to simulate a different machine
	configPath, _ := config.GetConfigPath()
	os.Remove(configPath)

	// Try registering the same username again - server should reject
	err = registerUserAtURL(username, server.URL+"/register")
	if err == nil {
		t.Fatal("expected error for duplicate username, got nil")
	}

	if err.Error() != "failed to register user: Username already registered" {
		t.Errorf("error = %q, want %q", err.Error(), "failed to register user: Username already registered")
	}
}

// TestRegisterIntegration_InvalidUsername tests that invalid usernames
// are rejected by the server with a descriptive error
func TestRegisterIntegration_InvalidUsername(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(origHome, ".kube", "config")
	}
	os.Setenv("KUBECONFIG", kubeconfig)

	client := getIntegrationTestClient(t)
	handler := registration.NewRegistrationHandler(client)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}))
	defer server.Close()

	invalidUsernames := []string{
		"ab",        // too short
		"ALICE",     // uppercase
		"alice_bob", // special character
		"1alice",    // starts with number
		"system",    // reserved
	}

	for _, uname := range invalidUsernames {
		t.Run(uname, func(t *testing.T) {
			err := registerUserAtURL(uname, server.URL+"/register")
			if err == nil {
				t.Errorf("expected error for invalid username %q, got nil", uname)
			}
		})
	}
}

// TestRegisterIntegration_TokenIsValid tests that the token returned by
// registration can be used to create a K8s client
func TestRegisterIntegration_TokenIsValid(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(origHome, ".kube", "config")
	}
	os.Setenv("KUBECONFIG", kubeconfig)

	client := getIntegrationTestClient(t)
	ensureTestSystemRBAC(t, client)

	handler := registration.NewRegistrationHandler(client)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	}))
	defer server.Close()

	username := uniqueTestUsername()
	defer cleanupTestNamespace(t, client, username)
	defer cleanupTestClusterRoleBinding(t, client, username)

	// Register and get the token
	reqBody, _ := json.Marshal(RegisterRequest{Username: username})
	resp, err := http.Post(server.URL+"/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("POST error = %v", err)
	}
	defer resp.Body.Close()

	var regResp RegisterResponse
	json.NewDecoder(resp.Body).Decode(&regResp)

	if regResp.Token == "" {
		t.Fatal("token is empty")
	}

	// Verify the token can construct a valid client
	restConfig, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
	tokenClient, err := k8s.NewClientFromToken(regResp.Token, restConfig.Host)
	if err != nil {
		t.Fatalf("NewClientFromToken() error = %v", err)
	}

	if tokenClient == nil {
		t.Fatal("NewClientFromToken() returned nil")
	}
}
