package k8s

import (
	"context"
	"fmt"
	"slices"

	"github.com/baighasan/kubecraft/internal/config"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateServiceAccount(username string) error {
	// Create service account object
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      username,
			Namespace: c.namespace,
			Labels: map[string]string{
				"app":  config.CommonLabelValue,
				"user": username,
			},
		},
	}

	// Create the service account for the user
	_, err := c.clientset.
		CoreV1().
		ServiceAccounts(c.namespace).
		Create(
			context.TODO(),
			sa,
			metav1.CreateOptions{},
		)
	if err != nil {
		return fmt.Errorf("could not create ServiceAccount: %w", err)
	}

	return nil
}

func (c *Client) CreateRole() error {
	// Create role object
	r := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.UserRoleName,
			Namespace: c.namespace,
			Labels: map[string]string{
				"app":       config.CommonLabelValue,
				"component": "rbac", // Add to constants later to remove hardcoding
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumeclaims", "services"},
				Verbs:     []string{"get", "list", "create", "update", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods/logs"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"statefulsets"},
				Verbs:     []string{"create", "get", "list", "patch", "update", "delete"},
			},
		},
	}

	// Create role in cluster
	_, err := c.clientset.
		RbacV1().
		Roles(c.namespace).
		Create(
			context.TODO(),
			r,
			metav1.CreateOptions{},
		)
	if err != nil {
		return fmt.Errorf("could not create Role: %w", err)
	}

	return nil
}

func (c *Client) CreateRoleBinding(username string) error {
	// Create role binding object
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "binding-" + username, // Add to constants.go later to prevent hardcoding
			Namespace: c.namespace,
			Labels: map[string]string{
				"app":       config.CommonLabelValue,
				"component": "rbac",
				"user":      username,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      username,
				Namespace: c.namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     config.UserRoleName,
		},
	}

	// Create role binding in cluster
	_, err := c.clientset.
		RbacV1().
		RoleBindings(c.namespace).
		Create(
			context.TODO(),
			rb,
			metav1.CreateOptions{},
		)
	if err != nil {
		return fmt.Errorf("could not create RoleBinding: %w", err)
	}

	return nil
}

func (c *Client) CreateResourceQuota(username string) error {
	// Create resource quota object
	rq := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mc-compute-resources", // Add to constants.go later to prevent hardcoding
			Namespace: c.namespace,
			Labels: map[string]string{
				"app":  config.CommonLabelValue,
				"user": username,
			},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceRequestsCPU:            resource.MustParse("1500m"),
				corev1.ResourceRequestsMemory:         resource.MustParse("1536Mi"),
				corev1.ResourceLimitsCPU:              resource.MustParse("2250m"),
				corev1.ResourceLimitsMemory:           resource.MustParse("3Gi"),
				corev1.ResourcePersistentVolumeClaims: resource.MustParse("1"),
			},
		},
	}

	// Create resource quota in cluster
	_, err := c.clientset.
		CoreV1().
		ResourceQuotas(c.namespace).
		Create(
			context.TODO(),
			rq,
			metav1.CreateOptions{},
		)
	if err != nil {
		return fmt.Errorf("could not create ResourceQuota: %w", err)
	}

	return nil
}

func (c *Client) AddUserToCapacityChecker(username string) error {
	// Get the cluster role binding from the cluster
	crb, err := c.clientset.
		RbacV1().
		ClusterRoleBindings().
		Get(
			context.TODO(),
			config.CapacityCheckerBinding,
			metav1.GetOptions{},
		)
	if errors.IsNotFound(err) {
		return fmt.Errorf("could not find ClusterRoleBinding %s", config.CapacityCheckerBinding)
	}
	if err != nil {
		return fmt.Errorf("could not get ClusterRoleBinding %s", config.CapacityCheckerBinding)
	}

	// Build new subject object
	newSubject := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      username,
		Namespace: c.namespace,
	}

	// Check duplicate then append subject field in cluster role binding to include new user
	if slices.Contains(crb.Subjects, newSubject) {
		return fmt.Errorf("user already exists in cluster role binding")
	}
	crb.Subjects = append(crb.Subjects, newSubject)

	// Update clientset with new cluster role binding
	_, err = c.clientset.
		RbacV1().
		ClusterRoleBindings().
		Update(
			context.TODO(),
			crb,
			metav1.UpdateOptions{},
		)
	if err != nil {
		return fmt.Errorf("could not update ClusterRoleBinding %s: %w", config.CapacityCheckerClusterRole, err)
	}

	return nil
}

func (c *Client) RemoveUserFromCapacityChecker(username string) error {
	crb, err := c.clientset.
		RbacV1().
		ClusterRoleBindings().
		Get(
			context.TODO(),
			config.CapacityCheckerBinding,
			metav1.GetOptions{},
		)
	if err != nil {
		return fmt.Errorf("could not get ClusterRoleBinding %s: %w", config.CapacityCheckerBinding, err)
	}

	// Filter out the user's subject
	filtered := make([]rbacv1.Subject, 0, len(crb.Subjects))
	for _, s := range crb.Subjects {
		if s.Name == username && s.Namespace == c.namespace {
			continue
		}
		filtered = append(filtered, s)
	}
	crb.Subjects = filtered

	_, err = c.clientset.
		RbacV1().
		ClusterRoleBindings().
		Update(
			context.TODO(),
			crb,
			metav1.UpdateOptions{},
		)
	if err != nil {
		return fmt.Errorf("could not update ClusterRoleBinding %s: %w", config.CapacityCheckerBinding, err)
	}

	return nil
}
