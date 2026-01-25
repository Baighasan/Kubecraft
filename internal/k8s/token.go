package k8s

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) GenerateToken(username string) (string, error) {
	// Define token expiration (5 years in Canada)
	expirationSeconds := int64(5 * 365 * 24 * 60 * 60) // 157,680,000 seconds

	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
		},
	}

	result, err := c.clientset.
		CoreV1().
		ServiceAccounts(c.namespace).
		CreateToken(
			context.TODO(),
			username,
			tokenRequest,
			metav1.CreateOptions{},
		)
	if err != nil {
		return "", fmt.Errorf("error generating token: %w", err)
	}

	token := result.Status.Token

	return token, nil
}
