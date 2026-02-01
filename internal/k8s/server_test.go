//go:build integration

package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/baighasan/kubecraft/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServerExists_ReturnsFalse(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	exists, err := client.ServerExists("nonexistent")
	if err != nil {
		t.Fatalf("ServerExists() error = %v", err)
	}
	if exists {
		t.Error("ServerExists() = true, want false for nonexistent server")
	}
}

func TestServerExists_ReturnsTrue(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	port, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() error = %v", err)
	}

	err = client.CreateServer("testserver", username, port)
	if err != nil {
		t.Fatalf("CreateServer() error = %v", err)
	}

	exists, err := client.ServerExists("testserver")
	if err != nil {
		t.Fatalf("ServerExists() error = %v", err)
	}
	if !exists {
		t.Error("ServerExists() = false, want true for existing server")
	}
}

func TestAllocateNodePort_ReturnsPortInRange(t *testing.T) {
	client := GetTestClient(t)

	port, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() error = %v", err)
	}

	if port < int32(config.McNodePortRangeMin) || port > int32(config.McNodePortRangeMax) {
		t.Errorf("AllocateNodePort() = %d, want port in range %d-%d", port, config.McNodePortRangeMin, config.McNodePortRangeMax)
	}
}

func TestAllocateNodePort_SkipsOccupiedPorts(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	// Allocate first port and create a server on it
	port1, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() first call error = %v", err)
	}

	err = client.CreateServer("server1", username, port1)
	if err != nil {
		t.Fatalf("CreateServer() error = %v", err)
	}

	// Allocate second port — should be different
	port2, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() second call error = %v", err)
	}

	if port1 == port2 {
		t.Errorf("AllocateNodePort() returned same port %d twice", port1)
	}
}

func TestCreateServer_Success(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	port, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() error = %v", err)
	}

	err = client.CreateServer("testserver", username, port)
	if err != nil {
		t.Fatalf("CreateServer() error = %v", err)
	}

	// Verify StatefulSet exists
	exists, err := client.ServerExists("testserver")
	if err != nil {
		t.Fatalf("ServerExists() error = %v", err)
	}
	if !exists {
		t.Error("StatefulSet was not created")
	}

	// Verify Service exists with correct NodePort
	svc, err := client.clientset.CoreV1().Services(client.namespace).Get(
		context.TODO(),
		"testserver",
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to get Service: %v", err)
	}
	if svc.Spec.Ports[0].NodePort != port {
		t.Errorf("Service NodePort = %d, want %d", svc.Spec.Ports[0].NodePort, port)
	}
}

func TestCreateServer_DuplicateNameFails(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	port, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() error = %v", err)
	}

	err = client.CreateServer("testserver", username, port)
	if err != nil {
		t.Fatalf("First CreateServer() error = %v", err)
	}

	port2, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() error = %v", err)
	}

	err = client.CreateServer("testserver", username, port2)
	if err == nil {
		t.Error("Second CreateServer() expected error for duplicate name, got nil")
	}
}

func TestDeleteServer_Success(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	port, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() error = %v", err)
	}

	err = client.CreateServer("testserver", username, port)
	if err != nil {
		t.Fatalf("CreateServer() error = %v", err)
	}

	err = client.DeleteServer("testserver")
	if err != nil {
		t.Fatalf("DeleteServer() error = %v", err)
	}

	// Verify server no longer exists
	exists, err := client.ServerExists("testserver")
	if err != nil {
		t.Fatalf("ServerExists() error = %v", err)
	}
	if exists {
		t.Error("Server still exists after deletion")
	}
}

func TestDeleteServer_NonexistentFails(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	err := client.DeleteServer("nonexistent")
	if err == nil {
		t.Error("DeleteServer() expected error for nonexistent server, got nil")
	}
}

func TestListServers_Empty(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	servers, err := client.ListServers()
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 0 {
		t.Errorf("ListServers() returned %d servers, want 0", len(servers))
	}
}

func TestListServers_ReturnsCreatedServer(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	port, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() error = %v", err)
	}

	err = client.CreateServer("testserver", username, port)
	if err != nil {
		t.Fatalf("CreateServer() error = %v", err)
	}

	servers, err := client.ListServers()
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("ListServers() returned %d servers, want 1", len(servers))
	}

	if servers[0].Name != "testserver" {
		t.Errorf("server Name = %q, want %q", servers[0].Name, "testserver")
	}
	if servers[0].NodePort != port {
		t.Errorf("server NodePort = %d, want %d", servers[0].NodePort, port)
	}
	if servers[0].Age.IsZero() {
		t.Error("server Age is zero")
	}
}

func TestScaleServer_StopAndStart(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	port, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() error = %v", err)
	}

	err = client.CreateServer("testserver", username, port)
	if err != nil {
		t.Fatalf("CreateServer() error = %v", err)
	}

	// Scale to 0 (stop)
	err = client.ScaleServer("testserver", 0)
	if err != nil {
		t.Fatalf("ScaleServer(0) error = %v", err)
	}

	// Verify status is stopped
	servers, err := client.ListServers()
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("ListServers() returned %d servers, want 1", len(servers))
	}
	if servers[0].Status != "stopped" {
		t.Errorf("server Status = %q, want %q after scaling to 0", servers[0].Status, "stopped")
	}

	// Scale to 1 (start)
	err = client.ScaleServer("testserver", 1)
	if err != nil {
		t.Fatalf("ScaleServer(1) error = %v", err)
	}

	// Verify status is running
	servers, err = client.ListServers()
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if servers[0].Status != "running" {
		t.Errorf("server Status = %q, want %q after scaling to 1", servers[0].Status, "running")
	}
}

func TestScaleServer_InvalidReplicas(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	err := client.ScaleServer("testserver", 2)
	if err == nil {
		t.Error("ScaleServer(2) expected error, got nil")
	}

	err = client.ScaleServer("testserver", -1)
	if err == nil {
		t.Error("ScaleServer(-1) expected error, got nil")
	}
}

func TestScaleServer_NonexistentFails(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	err := client.ScaleServer("nonexistent", 1)
	if err == nil {
		t.Error("ScaleServer() expected error for nonexistent server, got nil")
	}
}

func TestCheckNodeCapacity_PassesWhenEmpty(t *testing.T) {
	client := GetTestClient(t)

	err := client.CheckNodeCapacity()
	if err != nil {
		t.Errorf("CheckNodeCapacity() error = %v, want nil when no servers running", err)
	}
}

func TestListServers_ShowsStoppedServer(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	port, err := client.AllocateNodePort()
	if err != nil {
		t.Fatalf("AllocateNodePort() error = %v", err)
	}

	err = client.CreateServer("testserver", username, port)
	if err != nil {
		t.Fatalf("CreateServer() error = %v", err)
	}

	// Stop the server
	err = client.ScaleServer("testserver", 0)
	if err != nil {
		t.Fatalf("ScaleServer(0) error = %v", err)
	}

	// Stopped server should still appear in list
	servers, err := client.ListServers()
	if err != nil {
		t.Fatalf("ListServers() error = %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("ListServers() returned %d servers, want 1", len(servers))
	}
	if servers[0].Status != "stopped" {
		t.Errorf("server Status = %q, want %q", servers[0].Status, "stopped")
	}
}

func TestWaitForReady_TimeoutOnNonexistent(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	CreateTestNamespace(t, client, username)
	defer CleanupNamespace(t, client, username)

	client.namespace = config.NamespacePrefix + username

	// Should timeout since no server exists
	err := client.WaitForReady("nonexistent", 10*time.Second)
	if err == nil {
		t.Error("WaitForReady() expected timeout error, got nil")
	}
}
