package server

import (
	"fmt"
	"os"
	"unicode"

	"github.com/baighasan/kubecraft/internal/cli"
	"github.com/baighasan/kubecraft/internal/config"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <server-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Create a Minecraft server",
	Long:  "I'll think of this later",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		return execute(serverName)
	},
}

func execute(serverName string) error {
	// Validate server name
	if err := ValidateServerName(serverName); err != nil {
		return fmt.Errorf("invalid server name: %w", err)
	}

	// Check if server already exists
	serverExists, err := cli.K8sClient.ServerExists(serverName)
	if err != nil {
		return fmt.Errorf("cannot check server existence: %w", err)
	}
	if serverExists {
		return fmt.Errorf("server %s already exists", serverName)
	}

	// Run pre-flight checks
	err = cli.K8sClient.CheckNodeCapacity()
	if err != nil {
		return err // returning error to send correct message to user, unsure if this is best practice
	}

	// Get available nodeport
	port, err := cli.K8sClient.AllocateNodePort()
	if err != nil {
		return fmt.Errorf("cannot allocate node port: %w", err)
	}

	// Create Minecraft server
	err = cli.K8sClient.CreateServer(serverName, cli.AppConfig.Username, port)
	if err != nil {
		return fmt.Errorf("cannot create server: %w", err)
	}

	// Wait for pod to be ready
	err = cli.K8sClient.WaitForReady(serverName)
	if err != nil {
		return fmt.Errorf("server %s unable to start: %w", serverName, err)
	}

	fmt.Fprintf(os.Stderr, "Server %s is ready at %s:%d\n", serverName, config.ClusterEndpoint, port)

	return nil
}

func ValidateServerName(name string) error {
	// Check length
	if len(name) < config.MinServerNameLength || len(name) > config.MaxServerNameLength {
		return fmt.Errorf("server name must be between %d and %d characters", config.MinServerNameLength, config.MaxServerNameLength)
	}

	// Check name is only lowercase letters and digits
	for _, r := range name {
		if !unicode.IsLower(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("server name must contain only lowercase letters and numbers")
		}
	}

	// Check first letter is lowercase
	if !unicode.IsLower(rune(name[0])) {
		return fmt.Errorf("server name must start with a lowercase letter")
	}

	return nil
}

func init() {
	serverCmd.AddCommand(createCmd)
}
