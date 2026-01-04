package k8s

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/baighasan/kubecraft/pkg/config"
)

func TestGenerateToken_Success(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace and ServiceAccount
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	err = client.CreateServiceAccount(username)
	if err != nil {
		t.Fatalf("CreateServiceAccount() error = %v", err)
	}

	// Wait for ServiceAccount to be ready
	WaitForServiceAccount(t, client, config.NamespacePrefix+username, username)

	// Generate token
	token, err := client.GenerateToken(username)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Verify token is not empty
	if token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	// Verify token is a valid JWT format (3 parts separated by dots)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("Token has %d parts, want 3 (JWT format)", len(parts))
	}
}

func TestGenerateToken_ValidJWT(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace and ServiceAccount
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	err = client.CreateServiceAccount(username)
	if err != nil {
		t.Fatalf("CreateServiceAccount() error = %v", err)
	}

	// Wait for ServiceAccount to be ready
	WaitForServiceAccount(t, client, config.NamespacePrefix+username, username)

	// Generate token
	token, err := client.GenerateToken(username)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Parse JWT payload (middle part)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("Invalid JWT format")
	}

	// Decode payload (base64url)
	payload := parts[1]
	// Add padding if needed
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}

	payloadBytes, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("Failed to decode JWT payload: %v", err)
	}

	// Parse JSON payload
	var claims map[string]interface{}
	err = json.Unmarshal(payloadBytes, &claims)
	if err != nil {
		t.Fatalf("Failed to parse JWT claims: %v", err)
	}

	// Verify claims contain expected fields
	expectedFields := []string{"exp", "iat", "iss", "sub"}
	for _, field := range expectedFields {
		if _, exists := claims[field]; !exists {
			t.Errorf("JWT missing claim %q", field)
		}
	}

	// Verify expiration is set (should be ~5 years from now)
	if exp, ok := claims["exp"].(float64); ok {
		if exp == 0 {
			t.Error("JWT expiration is 0")
		}
	} else {
		t.Error("JWT exp claim is not a number")
	}
}

func TestGenerateToken_NonexistentServiceAccount(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace but NOT ServiceAccount
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	// Try to generate token - should fail
	token, err := client.GenerateToken(username)
	if err == nil {
		t.Fatal("GenerateToken() expected error for nonexistent ServiceAccount, got nil")
	}

	if token != "" {
		t.Errorf("GenerateToken() returned token %q on error, want empty string", token)
	}
}

func TestGenerateToken_ExpirationCorrect(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace and ServiceAccount
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	err = client.CreateServiceAccount(username)
	if err != nil {
		t.Fatalf("CreateServiceAccount() error = %v", err)
	}

	// Wait for ServiceAccount to be ready
	WaitForServiceAccount(t, client, config.NamespacePrefix+username, username)

	// Generate token
	token, err := client.GenerateToken(username)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Parse JWT to check expiration
	parts := strings.Split(token, ".")
	payload := parts[1]
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}

	payloadBytes, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("Failed to decode JWT payload: %v", err)
	}

	var claims map[string]interface{}
	err = json.Unmarshal(payloadBytes, &claims)
	if err != nil {
		t.Fatalf("Failed to parse JWT claims: %v", err)
	}

	// Calculate expected expiration duration
	// exp and iat are Unix timestamps (seconds since epoch)
	exp, expOk := claims["exp"].(float64)
	iat, iatOk := claims["iat"].(float64)

	if !expOk || !iatOk {
		t.Fatal("JWT missing exp or iat claims")
	}

	// Calculate duration in seconds
	duration := int64(exp - iat)

	// Should be approximately 5 years (allow 1 day tolerance for clock skew)
	fiveYears := int64(config.TokenExpirySeconds)
	tolerance := int64(24 * 60 * 60) // 1 day

	if duration < fiveYears-tolerance || duration > fiveYears+tolerance {
		t.Errorf("Token expiration duration = %d seconds (~%d years), want ~%d seconds (5 years)",
			duration, duration/(365*24*60*60), fiveYears)
	}
}
