package k8s

import (
	"context"
	"fmt"

	"github.com/baighasan/kubecraft/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateNamespace(username string) error {
	// Build namespace name
	nsName := config.NamespacePrefix + username

	// Build namespace object
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nsName,
			Labels: map[string]string{
				"app":  config.CommonLabelValue,
				"user": username,
			},
		},
	}

	// Create namespace
	_, err := c.clientset.
		CoreV1().
		Namespaces().
		Create(
			context.TODO(),
			ns,
			metav1.CreateOptions{},
		)
	if errors.IsAlreadyExists(err) {
		return fmt.Errorf("namespace already exists")
	}
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Set namespace once namespace has been successfully created
	c.namespace = nsName
	return nil
}

func (c *Client) NamespaceExists(username string) (bool, error) {
	// Build namespace name
	nsName := "mc-" + username

	// Check if namespace exists
	_, err := c.clientset.
		CoreV1().
		Namespaces().
		Get(
			context.TODO(),
			nsName,
			metav1.GetOptions{},
		)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error getting namespace: %w", err)
	}

	return true, nil
}

func (c *Client) CountUserNamespaces() (int, error) {
	// Get all the existing namespaces as a list
	nsList, err := c.clientset.
		CoreV1().
		Namespaces().
		List(
			context.TODO(),
			metav1.ListOptions{
				LabelSelector: config.CommonLabelSelector,
			},
		)
	if err != nil {
		return 0, fmt.Errorf("error getting namespaces: %w", err)
	}

	// Count number of namespaces
	nsNum := len(nsList.Items)

	return nsNum, nil
}
