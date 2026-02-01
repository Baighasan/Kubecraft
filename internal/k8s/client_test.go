//go:build integration

package k8s

import (
	"testing"
)

func TestNewInClusterClient_OutsideCluster(t *testing.T) {
	// This test runs outside a cluster, so it should fail
	client, err := NewInClusterClient()

	if err == nil {
		t.Fatal("NewInClusterClient() expected error outside cluster, got nil")
	}

	if client != nil {
		t.Error("NewInClusterClient() expected nil client on error, got non-nil")
	}

	// Verify error message is informative
	if err.Error() == "" {
		t.Error("NewInClusterClient() error message is empty")
	}
}

func TestNewClientFromToken(t *testing.T) {
	// NewClientFromToken should create a client without error given valid inputs
	// (it won't connect, but it should construct the client)
	client, err := NewClientFromToken("fake-token", "127.0.0.1:6443", "testuser")
	if err != nil {
		t.Fatalf("NewClientFromToken() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewClientFromToken() returned nil client")
	}

	if client.clientset == nil {
		t.Error("client.clientset is nil")
	}

	expectedNamespace := "mc-testuser"
	if client.namespace != expectedNamespace {
		t.Errorf("client.namespace = %q, want %q", client.namespace, expectedNamespace)
	}
}

func TestClient_GetClientset(t *testing.T) {
	client := GetTestClient(t)

	clientset := client.GetClientset()
	if clientset == nil {
		t.Error("GetClientset() returned nil")
	}

	// Verify clientset is functional (can make API call)
	_, err := clientset.Discovery().ServerVersion()
	if err != nil {
		t.Errorf("GetClientset() clientset cannot connect to API server: %v", err)
	}
}
