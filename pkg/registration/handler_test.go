package registration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baighasan/kubecraft/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Test HTTP method validation
func TestHandler_MethodNotAllowed(t *testing.T) {
	client := getTestClient(t)
	handler := NewRegistrationHandler(client)

	// Test GET request (should fail)
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET request status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}

	var response RegisterResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("response.Status = %q, want %q", response.Status, "error")
	}
}

// Test invalid JSON parsing
func TestHandler_InvalidJSON(t *testing.T) {
	client := getTestClient(t)
	handler := NewRegistrationHandler(client)

	// Send malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("not valid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var response RegisterResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("response.Status = %q, want %q", response.Status, "error")
	}
}

// Test username validation integration
func TestHandler_InvalidUsername(t *testing.T) {
	client := getTestClient(t)
	handler := NewRegistrationHandler(client)

	invalidUsernames := []string{
		"ab",        // too short
		"ALICE",     // uppercase
		"alice_bob", // special character
		"1alice",    // starts with number
		"system",    // reserved
	}

	for _, username := range invalidUsernames {
		t.Run(username, func(t *testing.T) {
			reqBody := RegisterRequest{Username: username}
			body, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Invalid username %q status = %d, want %d", username, w.Code, http.StatusBadRequest)
			}

			var response RegisterResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Status != "error" {
				t.Errorf("response.Status = %q, want %q", response.Status, "error")
			}
		})
	}
}

// Test successful registration
func TestHandler_SuccessfulRegistration(t *testing.T) {
	client := getTestClient(t)
	ensureSystemRBAC(t, client) // Ensure ClusterRole/Binding exist
	handler := NewRegistrationHandler(client)

	username := uniqueUsername()
	defer cleanupNamespace(t, client, username)
	defer cleanupClusterRoleBinding(t, client, username)

	reqBody := RegisterRequest{Username: username}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// Check HTTP status
	if w.Code != http.StatusCreated {
		t.Errorf("Registration status = %d, want %d. Response: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Parse response
	var response RegisterResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response fields
	if response.Status != "success" {
		t.Errorf("response.Status = %q, want %q", response.Status, "success")
	}

	if response.Username != username {
		t.Errorf("response.Username = %q, want %q", response.Username, username)
	}

	if response.Token == "" {
		t.Error("response.Token is empty, expected a valid token")
	}

	// Verify namespace was created
	exists, err := client.NamespaceExists(username)
	if err != nil {
		t.Fatalf("Failed to check namespace existence: %v", err)
	}
	if !exists {
		t.Errorf("Namespace for user %q was not created", username)
	}
}

// Test duplicate username rejection
func TestHandler_DuplicateUsername(t *testing.T) {
	client := getTestClient(t)
	ensureSystemRBAC(t, client)
	handler := NewRegistrationHandler(client)

	username := uniqueUsername()
	defer cleanupNamespace(t, client, username)
	defer cleanupClusterRoleBinding(t, client, username)

	// First registration (should succeed)
	reqBody := RegisterRequest{Username: username}
	body, _ := json.Marshal(reqBody)

	req1 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()

	handler(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("First registration failed with status %d", w1.Code)
	}

	// Second registration (should fail)
	req2 := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	handler(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Errorf("Duplicate registration status = %d, want %d", w2.Code, http.StatusConflict)
	}

	var response RegisterResponse
	if err := json.NewDecoder(w2.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("response.Status = %q, want %q", response.Status, "error")
	}
}

// Test JSON response format
func TestHandler_ResponseFormat(t *testing.T) {
	client := getTestClient(t)
	ensureSystemRBAC(t, client)
	handler := NewRegistrationHandler(client)

	username := uniqueUsername()
	defer cleanupNamespace(t, client, username)
	defer cleanupClusterRoleBinding(t, client, username)

	reqBody := RegisterRequest{Username: username}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	// Check Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	// Verify JSON is valid
	var response RegisterResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}
}

// Test that all K8s resources are created
func TestHandler_CreatesAllResources(t *testing.T) {
	client := getTestClient(t)
	ensureSystemRBAC(t, client)
	handler := NewRegistrationHandler(client)

	username := uniqueUsername()
	defer cleanupNamespace(t, client, username)
	defer cleanupClusterRoleBinding(t, client, username)

	reqBody := RegisterRequest{Username: username}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Registration failed with status %d: %s", w.Code, w.Body.String())
	}

	ctx := context.Background()
	nsName := config.NamespacePrefix + username

	// Verify Namespace
	ns, err := client.GetClientset().CoreV1().Namespaces().Get(
		ctx,
		nsName,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Errorf("Namespace not created: %v", err)
	}
	if ns.Labels[config.CommonLabelKey] != config.CommonLabelValue {
		t.Errorf("Namespace missing label %s=%s", config.CommonLabelKey, config.CommonLabelValue)
	}

	// Verify ServiceAccount
	_, err = client.GetClientset().CoreV1().ServiceAccounts(nsName).Get(
		ctx,
		username,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Errorf("ServiceAccount not created: %v", err)
	}

	// Verify Role
	_, err = client.GetClientset().RbacV1().Roles(nsName).Get(
		ctx,
		config.UserRoleName,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Errorf("Role not created: %v", err)
	}

	// Verify RoleBinding
	_, err = client.GetClientset().RbacV1().RoleBindings(nsName).Get(
		ctx,
		"binding-"+username,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Errorf("RoleBinding not created: %v", err)
	}

	// Verify ResourceQuota
	_, err = client.GetClientset().CoreV1().ResourceQuotas(nsName).Get(
		ctx,
		"mc-compute-resources",
		metav1.GetOptions{},
	)
	if err != nil {
		t.Errorf("ResourceQuota not created: %v", err)
	}
}
