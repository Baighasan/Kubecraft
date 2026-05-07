package k8s

import (
	"context"
	"fmt"

	"github.com/baighasan/kubecraft/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidateControlPlaneRBAC checks that the static control-plane RBAC resources
// required by the registration service exist. It should be called at startup.
func (c *Client) ValidateControlPlaneRBAC(ctx context.Context) error {
	required := []struct {
		name string
		kind string
		get  func() error
	}{
		{
			name: config.CapacityCheckerClusterRole,
			kind: "ClusterRole",
			get: func() error {
				_, err := c.clientset.RbacV1().ClusterRoles().Get(ctx, config.CapacityCheckerClusterRole, metav1.GetOptions{})
				return err
			},
		},
		{
			name: config.CapacityCheckerBinding,
			kind: "ClusterRoleBinding",
			get: func() error {
				_, err := c.clientset.RbacV1().ClusterRoleBindings().Get(ctx, config.CapacityCheckerBinding, metav1.GetOptions{})
				return err
			},
		},
		{
			name: config.RegistrationClusterRole,
			kind: "ClusterRole",
			get: func() error {
				_, err := c.clientset.RbacV1().ClusterRoles().Get(ctx, config.RegistrationClusterRole, metav1.GetOptions{})
				return err
			},
		},
		{
			name: config.RegistrationClusterRoleBinding,
			kind: "ClusterRoleBinding",
			get: func() error {
				_, err := c.clientset.RbacV1().ClusterRoleBindings().Get(ctx, config.RegistrationClusterRoleBinding, metav1.GetOptions{})
				return err
			},
		},
	}

	for _, r := range required {
		if err := r.get(); err != nil {
			return fmt.Errorf("required %s %q not found: %w", r.kind, r.name, err)
		}
	}

	return nil
}
