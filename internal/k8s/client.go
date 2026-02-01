package k8s

import (
	"fmt"

	"github.com/baighasan/kubecraft/internal/config"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	clientset *kubernetes.Clientset
	namespace string
}

func NewInClusterClient() (*Client, error) {
	// Get kubeconfig file within cluster
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting kubernetes config: %w", err)
	}

	// Get clientset using config to talk to kubernetes
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error getting kubernetes client: %w", err)
	}

	return &Client{
		clientset: clientset,
		namespace: "",
	}, nil
}

func NewClientFromToken(token string, endpoint string, username string) (*Client, error) {
	cfg := &rest.Config{
		Host:        "https://" + endpoint,
		BearerToken: token,
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error getting kubernetes client: %w", err)
	}

	client := &Client{
		clientset: clientset,
		namespace: config.NamespacePrefix + username,
	}

	return client, nil
}

// NewClientFromRestConfig creates a Client from an existing rest.Config.
// Useful for testing with kubeconfig-derived configurations.
func NewClientFromRestConfig(config *rest.Config) (*Client, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error getting kubernetes client: %w", err)
	}

	return &Client{
		clientset: clientset,
		namespace: "",
	}, nil
}

// GetClientset returns the underlying Kubernetes clientset
// Primarily used for testing and advanced operations
func (c *Client) GetClientset() *kubernetes.Clientset {
	return c.clientset
}
