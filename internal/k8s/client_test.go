package k8s

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewClientFromKubeConfig_Success(t *testing.T) {
	// Skip if no kubeconfig available
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := os.Getenv("HOME")
		if home == "" {
			t.Skip("Skipping test: no HOME environment variable")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		t.Skipf("Skipping test: kubeconfig not found at %s", kubeconfig)
	}

	client, err := NewClientFromKubeConfig(kubeconfig)
	if err != nil {
		t.Fatalf("NewClientFromKubeConfig() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewClientFromKubeConfig() returned nil client")
	}

	if client.clientset == nil {
		t.Error("client.clientset is nil")
	}

	if client.namespace != "" {
		t.Errorf("client.namespace = %q, want empty string", client.namespace)
	}
}

func TestNewClientFromKubeConfig_InvalidPath(t *testing.T) {
	invalidPath := "/nonexistent/path/to/kubeconfig"

	client, err := NewClientFromKubeConfig(invalidPath)
	if err == nil {
		t.Fatal("NewClientFromKubeConfig() expected error for invalid path, got nil")
	}

	if client != nil {
		t.Error("NewClientFromKubeConfig() expected nil client on error, got non-nil")
	}
}

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

func TestClient_GetClientset(t *testing.T) {
	// Skip if no kubeconfig available
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := os.Getenv("HOME")
		if home == "" {
			t.Skip("Skipping test: no HOME environment variable")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		t.Skipf("Skipping test: kubeconfig not found at %s", kubeconfig)
	}

	client, err := NewClientFromKubeConfig(kubeconfig)
	if err != nil {
		t.Fatalf("NewClientFromKubeConfig() error = %v", err)
	}

	clientset := client.GetClientset()
	if clientset == nil {
		t.Error("GetClientset() returned nil")
	}

	// Verify clientset is functional (can make API call)
	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		t.Errorf("GetClientset() clientset cannot connect to API server: %v", err)
	}
}
