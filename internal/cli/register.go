package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/baighasan/kubecraft/internal/config"
	"github.com/spf13/cobra"
)

// RegisterRequest represents the request to the registration service
type RegisterRequest struct {
	Username string `json:"username"`
}

// RegisterResponse represents what the registration service sends back
type RegisterResponse struct {
	Status   string `json:"status"`             // "success" or "error"
	Username string `json:"username,omitempty"` // only in success
	Token    string `json:"token,omitempty"`    // only in success
	Message  string `json:"message,omitempty"`  // only in error
}

var username string

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a user",
	Long:  "I'll think of this later",
	RunE: func(cmd *cobra.Command, args []string) error {
		return registerUser(username)
	},
}

func registerUser(username string) error {
	host, _, err := net.SplitHostPort(config.ClusterEndpoint)
	if err != nil {
		// No port in endpoint, use as-is
		host = config.ClusterEndpoint
	}
	url := fmt.Sprintf("http://%s:%d/register", host, config.RegistrationServicePort)
	return registerUserAtURL(username, url)
}

func registerUserAtURL(username string, url string) error {
	configExists, err := config.CheckConfigExists()
	if err != nil {
		return fmt.Errorf("failed to check existing config: %v", err)
	}
	if configExists {
		return fmt.Errorf("you are already registered. Delete ~/.kubecraft/config first if you want to re-register")
	}

	reqPayload := &RegisterRequest{Username: username}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("could not reach registration server at %s:%d: %v", config.ClusterEndpoint, config.RegistrationServicePort, err)
	}
	defer resp.Body.Close()

	var regResponse RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResponse); err != nil {
		return fmt.Errorf("registration server returned status %d and response could not be parsed", resp.StatusCode)
	}

	if resp.StatusCode >= 300 || regResponse.Status != "success" {
		return fmt.Errorf("failed to register user: %s", regResponse.Message)
	}

	cfg := &config.Config{
		Username: regResponse.Username,
		Token:    regResponse.Token,
	}

	err = config.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully registered user: %v. Configuration saved to ~/.kubecraft/config\n", regResponse.Username)

	return nil
}

func init() {
	registerCmd.Flags().StringVarP(&username, "username", "u", "", "Username to register")
	err := registerCmd.MarkFlagRequired("username")
	if err != nil {
		panic(err)
	}
	RootCmd.AddCommand(registerCmd)
}
