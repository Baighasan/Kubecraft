package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	clientset *kubernetes.Clientset
	namespace string
}

func NewInClusterClient() (*Client, error) {
	// Get kubeconfig file within cluster
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting kubernetes config: %w", err)
	}

	// Get clientset using config to talk to kubernetes
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error getting kubernetes client: %w", err)
	}

	return &Client{
		clientset: clientset,
		namespace: "",
	}, nil
}

func NewClientFromToken(token string, endpoint string) (*Client, error) {
	config := &rest.Config{
		Host:        "https://" + endpoint,
		BearerToken: token,
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error getting kubernetes client: %w", err)
	}

	client := &Client{
		clientset: clientset,
		namespace: "",
	}

	return client, nil
}

// GetClientset returns the underlying Kubernetes clientset
// Primarily used for testing and advanced operations
func (c *Client) GetClientset() *kubernetes.Clientset {
	return c.clientset
}
