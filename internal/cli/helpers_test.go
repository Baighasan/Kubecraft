//go:build integration

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/baighasan/kubecraft/internal/config"
	"github.com/baighasan/kubecraft/internal/k8s"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// getIntegrationTestClient creates a k8s client for integration tests
func getIntegrationTestClient(t *testing.T) *k8s.Client {
	t.Helper()

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home := os.Getenv("HOME")
		if home == "" {
			t.Fatal("HOME environment variable not set")
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		t.Fatalf("Failed to build config from kubeconfig: %v", err)
	}

	client, err := k8s.NewClientFromRestConfig(restConfig)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return client
}

// uniqueTestUsername generates a unique username for tests
func uniqueTestUsername() string {
	return fmt.Sprintf("testuser%d", time.Now().UnixNano()%1000000)
}

// cleanupTestNamespace deletes a namespace and all resources inside it
func cleanupTestNamespace(t *testing.T, client *k8s.Client, username string) {
	t.Helper()

	nsName := config.NamespacePrefix + username
	ctx := context.Background()

	err := client.GetClientset().CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
	if err != nil {
		t.Logf("Cleanup warning: %v", err)
	}

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
				return
			}
		}
	}
}

// cleanupTestClusterRoleBinding removes a subject from the capacity checker ClusterRoleBinding
func cleanupTestClusterRoleBinding(t *testing.T, client *k8s.Client, username string) {
	t.Helper()

	ctx := context.Background()
	nsName := config.NamespacePrefix + username

	crb, err := client.GetClientset().RbacV1().ClusterRoleBindings().Get(
		ctx,
		config.CapacityCheckerBinding,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Logf("ClusterRoleBinding cleanup warning: %v", err)
		return
	}

	newSubjects := []rbacv1.Subject{}
	for _, subject := range crb.Subjects {
		if subject.Namespace != nsName || subject.Name != username {
			newSubjects = append(newSubjects, subject)
		}
	}

	crb.Subjects = newSubjects

	_, err = client.GetClientset().RbacV1().ClusterRoleBindings().Update(
		ctx,
		crb,
		metav1.UpdateOptions{},
	)
	if err != nil {
		t.Logf("ClusterRoleBinding update warning: %v", err)
	}
}

// ensureTestSystemRBAC ensures the system ClusterRole and ClusterRoleBinding exist
func ensureTestSystemRBAC(t *testing.T, client *k8s.Client) {
	t.Helper()

	ctx := context.Background()

	_, err := client.GetClientset().RbacV1().ClusterRoles().Get(
		ctx,
		config.CapacityCheckerClusterRole,
		metav1.GetOptions{},
	)
	if err != nil {
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

	_, err = client.GetClientset().RbacV1().ClusterRoleBindings().Get(
		ctx,
		config.CapacityCheckerBinding,
		metav1.GetOptions{},
	)
	if err != nil {
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
			Subjects: []rbacv1.Subject{},
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
