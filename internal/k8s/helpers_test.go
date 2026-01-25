//go:build integration

package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/baighasan/kubecraft/internal/config"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetTestClient creates a k8s client for integration tests
// Uses KUBECONFIG env var or default kubeconfig location
func GetTestClient(t *testing.T) *Client {
	t.Helper()

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := os.Getenv("HOME")
		if home == "" {
			t.Fatal("HOME environment variable not set")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	client, err := NewClientFromKubeConfig(kubeconfig)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return client
}

// UniqueNamespaceName generates a unique namespace name for tests
// Format: mc-test-<timestamp>
func UniqueNamespaceName() string {
	return fmt.Sprintf("test-%d", time.Now().UnixNano())
}

// UniqueUsername generates a unique username for tests
func UniqueUsername() string {
	return fmt.Sprintf("testuser-%d", time.Now().UnixNano()%1000000)
}

// CleanupNamespace deletes a namespace and all resources inside it
// Safe to call even if namespace doesn't exist
func CleanupNamespace(t *testing.T, client *Client, username string) {
	t.Helper()

	nsName := config.NamespacePrefix + username
	ctx := context.Background()

	// Get the raw clientset to delete namespace
	// (Client struct doesn't expose DeleteNamespace method)
	err := client.GetClientset().CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
	if err != nil {
		// Ignore "not found" errors
		t.Logf("Cleanup warning: %v", err)
	}

	// Wait for namespace to be fully deleted (max 30 seconds)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Logf("Timeout waiting for namespace %s to delete", nsName)
			return
		case <-ticker.C:
			_, err := client.GetClientset().CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
			if err != nil {
				// Namespace is gone
				return
			}
		}
	}
}

// CleanupClusterRoleBinding removes a subject from a ClusterRoleBinding
// Used to clean up capacity checker binding after tests
func CleanupClusterRoleBinding(t *testing.T, client *Client, username string) {
	t.Helper()

	ctx := context.Background()
	nsName := config.NamespacePrefix + username

	// Get the ClusterRoleBinding
	crb, err := client.GetClientset().RbacV1().ClusterRoleBindings().Get(
		ctx,
		config.CapacityCheckerBinding,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Logf("ClusterRoleBinding cleanup warning: %v", err)
		return
	}

	// Remove the test user's subject
	newSubjects := []rbacv1.Subject{}
	for _, subject := range crb.Subjects {
		if subject.Namespace != nsName || subject.Name != username {
			newSubjects = append(newSubjects, subject)
		}
	}

	crb.Subjects = newSubjects

	// Update the ClusterRoleBinding
	_, err = client.GetClientset().RbacV1().ClusterRoleBindings().Update(
		ctx,
		crb,
		metav1.UpdateOptions{},
	)
	if err != nil {
		t.Logf("ClusterRoleBinding update warning: %v", err)
	}
}

// EnsureSystemRBAC ensures the system ClusterRole and ClusterRoleBinding exist
// Required for capacity checker tests
func EnsureSystemRBAC(t *testing.T, client *Client) {
	t.Helper()

	ctx := context.Background()

	// Check if ClusterRole exists, create if not
	_, err := client.GetClientset().RbacV1().ClusterRoles().Get(
		ctx,
		config.CapacityCheckerClusterRole,
		metav1.GetOptions{},
	)
	if err != nil {
		// ClusterRole doesn't exist, create it
		clusterRole := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: config.CapacityCheckerClusterRole,
				Labels: map[string]string{
					config.CommonLabelKey: config.CommonLabelValue,
				},
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"namespaces", "services", "pods"},
					Verbs:     []string{"get", "list"},
				},
			},
		}

		_, err = client.GetClientset().RbacV1().ClusterRoles().Create(
			ctx,
			clusterRole,
			metav1.CreateOptions{},
		)
		if err != nil {
			t.Fatalf("Failed to create ClusterRole: %v", err)
		}
	}

	// Check if ClusterRoleBinding exists, create if not
	_, err = client.GetClientset().RbacV1().ClusterRoleBindings().Get(
		ctx,
		config.CapacityCheckerBinding,
		metav1.GetOptions{},
	)
	if err != nil {
		// ClusterRoleBinding doesn't exist, create it
		clusterRoleBinding := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: config.CapacityCheckerBinding,
				Labels: map[string]string{
					config.CommonLabelKey: config.CommonLabelValue,
				},
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     config.CapacityCheckerClusterRole,
			},
			Subjects: []rbacv1.Subject{}, // Empty initially
		}

		_, err = client.GetClientset().RbacV1().ClusterRoleBindings().Create(
			ctx,
			clusterRoleBinding,
			metav1.CreateOptions{},
		)
		if err != nil {
			t.Fatalf("Failed to create ClusterRoleBinding: %v", err)
		}
	}
}

// CreateTestNamespace creates a simple namespace for testing
// Does NOT create RBAC or other resources (use client methods for that)
func CreateTestNamespace(t *testing.T, client *Client, username string) {
	t.Helper()

	ctx := context.Background()
	nsName := config.NamespacePrefix + username

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				config.CommonLabelKey: config.CommonLabelValue,
				"user":                username,
			},
		},
	}

	_, err := client.GetClientset().CoreV1().Namespaces().Create(
		ctx,
		ns,
		metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}
}

// WaitForServiceAccount waits for a ServiceAccount to be ready
// ServiceAccounts need time to generate default secrets
func WaitForServiceAccount(t *testing.T, client *Client, namespace, name string) {
	t.Helper()

	ctx := context.Background()
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for ServiceAccount %s/%s to be ready", namespace, name)
		case <-ticker.C:
			_, err := client.GetClientset().CoreV1().ServiceAccounts(namespace).Get(
				ctx,
				name,
				metav1.GetOptions{},
			)
			if err == nil {
				// Small additional delay to ensure token is generated
				time.Sleep(500 * time.Millisecond)
				return
			}
		}
	}
}
