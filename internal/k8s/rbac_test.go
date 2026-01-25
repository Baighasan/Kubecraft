package k8s

import (
	"context"
	"testing"

	"github.com/baighasan/kubecraft/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateServiceAccount_Success(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace first
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	// Create ServiceAccount
	err = client.CreateServiceAccount(username)
	if err != nil {
		t.Fatalf("CreateServiceAccount() error = %v", err)
	}

	// Verify ServiceAccount exists
	nsName := config.NamespacePrefix + username
	sa, err := client.GetClientset().CoreV1().ServiceAccounts(nsName).Get(
		context.Background(),
		username,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to get ServiceAccount: %v", err)
	}

	// Verify labels
	if sa.Labels[config.CommonLabelKey] != config.CommonLabelValue {
		t.Errorf("ServiceAccount missing app=kubecraft label")
	}
	if sa.Labels["user"] != username {
		t.Errorf("ServiceAccount missing user label")
	}
}

func TestCreateRole_Success(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace first
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	// Create Role
	err = client.CreateRole()
	if err != nil {
		t.Fatalf("CreateRole() error = %v", err)
	}

	// Verify Role exists
	nsName := config.NamespacePrefix + username
	role, err := client.GetClientset().RbacV1().Roles(nsName).Get(
		context.Background(),
		config.UserRoleName,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to get Role: %v", err)
	}

	// Verify role has required permissions
	hasStatefulSets := false
	hasPVCs := false
	hasPods := false

	for _, rule := range role.Rules {
		for _, resource := range rule.Resources {
			if resource == "statefulsets" {
				hasStatefulSets = true
			}
			if resource == "persistentvolumeclaims" {
				hasPVCs = true
			}
			if resource == "pods" {
				hasPods = true
			}
		}
	}

	if !hasStatefulSets {
		t.Error("Role missing statefulsets permissions")
	}
	if !hasPVCs {
		t.Error("Role missing persistentvolumeclaims permissions")
	}
	if !hasPods {
		t.Error("Role missing pods permissions")
	}
}

func TestCreateRoleBinding_Success(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace, ServiceAccount, and Role first
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	err = client.CreateServiceAccount(username)
	if err != nil {
		t.Fatalf("CreateServiceAccount() error = %v", err)
	}

	err = client.CreateRole()
	if err != nil {
		t.Fatalf("CreateRole() error = %v", err)
	}

	// Create RoleBinding
	err = client.CreateRoleBinding(username)
	if err != nil {
		t.Fatalf("CreateRoleBinding() error = %v", err)
	}

	// Verify RoleBinding exists
	nsName := config.NamespacePrefix + username
	rb, err := client.GetClientset().RbacV1().RoleBindings(nsName).Get(
		context.Background(),
		"binding-"+username,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to get RoleBinding: %v", err)
	}

	// Verify RoleBinding references correct Role
	if rb.RoleRef.Name != config.UserRoleName {
		t.Errorf("RoleBinding references %q, want %q", rb.RoleRef.Name, config.UserRoleName)
	}

	// Verify RoleBinding has correct Subject
	if len(rb.Subjects) != 1 {
		t.Fatalf("RoleBinding has %d subjects, want 1", len(rb.Subjects))
	}

	if rb.Subjects[0].Name != username {
		t.Errorf("RoleBinding subject name = %q, want %q", rb.Subjects[0].Name, username)
	}
	if rb.Subjects[0].Namespace != nsName {
		t.Errorf("RoleBinding subject namespace = %q, want %q", rb.Subjects[0].Namespace, nsName)
	}
}

func TestCreateResourceQuota_Success(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)

	// Create namespace first
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	// Create ResourceQuota
	err = client.CreateResourceQuota(username)
	if err != nil {
		t.Fatalf("CreateResourceQuota() error = %v", err)
	}

	// Verify ResourceQuota exists
	nsName := config.NamespacePrefix + username
	rq, err := client.GetClientset().CoreV1().ResourceQuotas(nsName).Get(
		context.Background(),
		"mc-compute-resources",
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to get ResourceQuota: %v", err)
	}

	// Verify limits
	expectedLimits := map[string]string{
		"requests.cpu":           "1500m",
		"requests.memory":        "1536Mi",
		"limits.cpu":             "2250m",
		"limits.memory":          "3Gi",
		"persistentvolumeclaims": "1",
	}

	for resourceName, expectedValue := range expectedLimits {
		actualQuantity, exists := rq.Spec.Hard[corev1.ResourceName(resourceName)]
		if !exists {
			t.Errorf("ResourceQuota missing limit for %q", resourceName)
			continue
		}

		actualValue := actualQuantity.String()
		if actualValue != expectedValue {
			t.Errorf("ResourceQuota %q = %q, want %q", resourceName, actualValue, expectedValue)
		}
	}
}

func TestAddUserToCapacityChecker_Success(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)
	defer CleanupClusterRoleBinding(t, client, username)

	// Ensure system RBAC exists
	EnsureSystemRBAC(t, client)

	// Create namespace and ServiceAccount first
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	err = client.CreateServiceAccount(username)
	if err != nil {
		t.Fatalf("CreateServiceAccount() error = %v", err)
	}

	// Get initial subject count
	crb, err := client.GetClientset().RbacV1().ClusterRoleBindings().Get(
		context.Background(),
		config.CapacityCheckerBinding,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to get ClusterRoleBinding: %v", err)
	}
	initialCount := len(crb.Subjects)

	// Add user to capacity checker
	err = client.AddUserToCapacityChecker(username)
	if err != nil {
		t.Fatalf("AddUserToCapacityChecker() error = %v", err)
	}

	// Verify user was added
	crb, err = client.GetClientset().RbacV1().ClusterRoleBindings().Get(
		context.Background(),
		config.CapacityCheckerBinding,
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to get updated ClusterRoleBinding: %v", err)
	}

	finalCount := len(crb.Subjects)
	if finalCount != initialCount+1 {
		t.Errorf("ClusterRoleBinding has %d subjects, want %d", finalCount, initialCount+1)
	}

	// Verify the user's subject exists
	nsName := config.NamespacePrefix + username
	found := false
	for _, subject := range crb.Subjects {
		if subject.Name == username && subject.Namespace == nsName {
			found = true
			break
		}
	}

	if !found {
		t.Error("User's ServiceAccount not found in ClusterRoleBinding subjects")
	}
}

func TestAddUserToCapacityChecker_Duplicate(t *testing.T) {
	client := GetTestClient(t)
	username := UniqueUsername()
	defer CleanupNamespace(t, client, username)
	defer CleanupClusterRoleBinding(t, client, username)

	// Ensure system RBAC exists
	EnsureSystemRBAC(t, client)

	// Create namespace and ServiceAccount
	err := client.CreateNamespace(username)
	if err != nil {
		t.Fatalf("CreateNamespace() error = %v", err)
	}

	err = client.CreateServiceAccount(username)
	if err != nil {
		t.Fatalf("CreateServiceAccount() error = %v", err)
	}

	// Add user first time
	err = client.AddUserToCapacityChecker(username)
	if err != nil {
		t.Fatalf("AddUserToCapacityChecker() first call error = %v", err)
	}

	// Try to add again - should fail
	err = client.AddUserToCapacityChecker(username)
	if err == nil {
		t.Fatal("AddUserToCapacityChecker() expected error for duplicate, got nil")
	}

	// Verify error message
	expectedMsg := "user already exists in cluster role binding"
	if err.Error() != expectedMsg {
		t.Errorf("AddUserToCapacityChecker() error = %q, want %q", err.Error(), expectedMsg)
	}
}
