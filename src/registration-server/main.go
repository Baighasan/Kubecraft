package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/baighasan/kubecraft/pkg/k8s"
	"github.com/baighasan/kubecraft/pkg/registration"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// Get the k8s client, first try in cluster (regular use case)
	k8sClient, err := k8s.NewInClusterClient()
	if err != nil {
		// If that doesn't work fall back to default path
		kubeconfigPath := homedir.HomeDir() + "/.kube/config"
		k8sClient, err = k8s.NewClientFromKubeConfig(kubeconfigPath)
		if err != nil {
			fmt.Printf("failed to create k8s client: %s\n", err)
			os.Exit(1)
		}
	}

	// Set up routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/register", registration.NewRegistrationHandler(k8sClient))

	// Start Server on port 8080
	fmt.Printf("Starting server on port 8080\n")
	err = http.ListenAndServe(":8080", nil)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		return
	}
}
