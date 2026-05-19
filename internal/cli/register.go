package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// RegistrationCredentials holds the credentials within the response from registration server
type RegistrationCredentials struct {
	Username string
	Token    string
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
	configExists, err := config.CheckConfigExists()
	if err != nil {
		return fmt.Errorf("failed to check existing config: %v", err)
	} else if !configExists {
		return fmt.Errorf("config does not exist. run kubecraft init --ip <clusterIP> first")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.CheckRegistered() {
		return fmt.Errorf("you are already registered. delete user and token fields in ~/.kubecraft/config to register again")
	}

	registrationURL, err := cfg.RegistrationEndpoint()
	if err != nil {
		return fmt.Errorf("failed to build registration url: %w", err)
	}

	registrationCreds, err := registerUserAtURL(username, registrationURL)
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}

	cfg.Username = registrationCreds.Username
	cfg.Token = registrationCreds.Token

	err = config.SaveConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully registered user: %v. Configuration saved to ~/.kubecraft/config\n", username)

	return nil
}

func registerUserAtURL(username string, registrationURL string) (*RegistrationCredentials, error) {
	registrationCreds := &RegistrationCredentials{}
	reqPayload := &RegisterRequest{Username: username}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return registrationCreds, fmt.Errorf("failed to marshal payload: %v", err)
	}

	resp, err := http.Post(registrationURL+"/register", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return registrationCreds, fmt.Errorf("could not reach registration server at %s: %w", registrationURL, err)
	}
	defer resp.Body.Close()

	var regResponse RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResponse); err != nil {
		return registrationCreds, fmt.Errorf("registration server returned status %d and response could not be parsed: %w", resp.StatusCode, err)
	}

	if resp.StatusCode >= 300 || regResponse.Status != "success" {
		return registrationCreds, fmt.Errorf("failed to register user: %s", regResponse.Message)
	}

	registrationCreds.Username = regResponse.Username
	registrationCreds.Token = regResponse.Token

	return registrationCreds, nil
}

func init() {
	registerCmd.Flags().StringVarP(&username, "username", "u", "", "Username to register")
	err := registerCmd.MarkFlagRequired("username")
	if err != nil {
		panic(err)
	}
	RootCmd.AddCommand(registerCmd)
}
