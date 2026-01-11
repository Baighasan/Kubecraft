package registration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/baighasan/kubecraft/pkg/config"
	"github.com/baighasan/kubecraft/pkg/k8s"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getTestClient creates a k8s client for integration tests
// Uses KUBECONFIG env var or default kubeconfig location
func getTestClient(t *testing.T) *k8s.Client {
	t.Helper()

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := os.Getenv("HOME")
		if home == "" {
			t.Fatal("HOME environment variable not set")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	client, err := k8s.NewClientFromKubeConfig(kubeconfig)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return client
}

// uniqueUsername generates a unique username for tests
func uniqueUsername() string {
	return fmt.Sprintf("testuser-%d", time.Now().UnixNano()%1000000)
}

// cleanupNamespace deletes a namespace and all resources inside it
func cleanupNamespace(t *testing.T, client *k8s.Client, username string) {
	t.Helper()

	nsName := config.NamespacePrefix + username
	ctx := context.Background()

	// Delete namespace
	err := client.GetClientset().CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
	if err != nil {
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

// cleanupClusterRoleBinding removes a subject from a ClusterRoleBinding
func cleanupClusterRoleBinding(t *testing.T, client *k8s.Client, username string) {
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

// ensureSystemRBAC ensures the system ClusterRole and ClusterRoleBinding exist
// Required for capacity checker tests
func ensureSystemRBAC(t *testing.T, client *k8s.Client) {
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
