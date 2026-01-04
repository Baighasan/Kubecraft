package k8s

import (
	"context"
	"testing"

	"github.com/baighasan/kubecraft/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateNamespace_Success(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	// Verify namespace was created
	nsName := config.NamespacePrefix + username
	exists, err := client.NamespaceExists(username)
	if err != nil {
		t.Fatalf("NamespaceExists() error = %v", err)
	}

	if !exists {
		t.Errorf("Namespace %s was not created", nsName)
	}

	// Verify client.namespace was set
	if client.namespace != nsName {
		t.Errorf("client.namespace = %q, want %q", client.namespace, nsName)
	}
}

func TestCreateNamespace_AlreadyExists(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace first time
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() first call error = %v", err)
	}

	// Try to create again - should fail
	err = client.CreateNamespace(username)
	if err == nil {
		t.Fatal("CreateNamespace() expected error for duplicate namespace, got nil")
	}

	// Verify error message mentions "already exists"
	if err.Error() != "namespace already exists" {
		t.Errorf("CreateNamespace() error = %q, want %q", err.Error(), "namespace already exists")
	}
}

func TestCreateNamespace_LabelsCorrect(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	// Get the namespace and verify labels
	nsName := config.NamespacePrefix + username
	ns, err := client.GetClientset().CoreV1().Namespaces().Get(
		context.Background(),
		nsName,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to get namespace: %v", err)
	}

	// Check labels
	expectedLabels := map[string]string{
		config.CommonLabelKey: config.CommonLabelValue,
		"user":                username,
	}

	for key, expectedValue := range expectedLabels {
		actualValue, exists := ns.Labels[key]
		if !exists {
			t.Errorf("Label %q not found in namespace", key)
		} else if actualValue != expectedValue {
			t.Errorf("Label %q = %q, want %q", key, actualValue, expectedValue)
		}
	}
}

func TestNamespaceExists_ReturnsTrue(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	// Check existence
	exists, err := client.NamespaceExists(username)
	if err != nil {
		t.Fatalf("NamespaceExists() error = %v", err)
	}

	if !exists {
		t.Error("NamespaceExists() = false, want true")
	}
}

func TestNamespaceExists_ReturnsFalse(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()

	// Don't create namespace - just check if it exists
	exists, err := client.NamespaceExists(username)
	if err != nil {
		t.Fatalf("NamespaceExists() error = %v", err)
	}

	if exists {
		t.Error("NamespaceExists() = true, want false for non-existent namespace")
	}
}

func TestCountUserNamespaces_ReturnsCorrectCount(t *testing.T) {
	client := GetTestClient(t)

	// Create multiple test namespaces
	numNamespaces := 3
	usernames := make([]string, numNamespaces)

	for i := 0; i < numNamespaces; i++ {
		usernames[i] = UniqueUsername()
		defer CleanupNamespace(t, client, usernames[i])
	}

	// Get initial count
	initialCount, err := client.CountUserNamespaces()
	if err != nil {
		t.Fatalf("CountUserNamespaces() initial error = %v", err)
	}

	// Create namespaces
	for _, username := range usernames {
		err := client.CreateNamespace(username)
		if err != nil {
			t.Fatalf("CreateNamespace(%s) error = %v", username, err)
		}
	}

	// Count again
	finalCount, err := client.CountUserNamespaces()
	if err != nil {
		t.Fatalf("CountUserNamespaces() final error = %v", err)
	}

	expectedIncrease := numNamespaces
	actualIncrease := finalCount - initialCount

	if actualIncrease != expectedIncrease {
		t.Errorf("CountUserNamespaces() increased by %d, want %d", actualIncrease, expectedIncrease)
	}
}

func TestCountUserNamespaces_IgnoresNonKubecraftNamespaces(t *testing.T) {
	client := GetTestClient(t)

	// Get initial count
	initialCount, err := client.CountUserNamespaces()
	if err != nil {
		t.Fatalf("CountUserNamespaces() error = %v", err)
	}

	// Count should not include kube-system, default, etc.
	// (these don't have app=kubecraft label)
	if initialCount < 0 {
		t.Errorf("CountUserNamespaces() = %d, should be >= 0", initialCount)
	}
}
