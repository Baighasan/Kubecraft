package server

import (
	"fmt"
	"os"
	"unicode"

	"github.com/baighasan/kubecraft/internal/cli"
	"github.com/baighasan/kubecraft/internal/config"
	"github.com/baighasan/kubecraft/internal/k8s"
	"github.com/spf13/cobra"
)

const serverImageEnvVar = "KUBECRAFT_SERVER_IMAGE"

var serverImage string

type createInput struct {
	ServerName string
	Version    string
	GameMode   string
	MaxPlayers int
}

var createCmd = &cobra.Command{
	Use:   "create [server-name]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Create a Minecraft server",
	Long:  "I'll think of this later",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return runCreateWizard()
		}

		input := buildDefaultCreateInput(args[0])
		return executeCreateWithInput(input)
	},
}

func runCreateWizard() error {
	return fmt.Errorf("wizard mode is not implemented yet")
}

func buildDefaultCreateInput(serverName string) createInput {
	return createInput{
		ServerName: serverName,
		Version:    config.DefaultMinecraftVersion,
		GameMode:   config.DefaultGameMode,
		MaxPlayers: config.DefaultMaxPlayers,
	}
}

func resolveServerImage(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	envValue := os.Getenv(serverImageEnvVar)
	if envValue != "" {
		return envValue
	}

	return ""
}

func executeCreateWithInput(input createInput) error {
	selectedImage := resolveServerImage(serverImage)

	// Validate server name
	if err := ValidateServerName(input.ServerName); err != nil {
		return fmt.Errorf("invalid server name: %w", err)
	}

	// Check if server already exists
	serverExists, err := cli.K8sClient.ServerExists(input.ServerName)
	if err != nil {
		return fmt.Errorf("cannot check server existence: %w", err)
	}
	if serverExists {
		return fmt.Errorf("server %s already exists", input.ServerName)
	}

	// Run pre-flight checks
	fmt.Fprintln(os.Stderr, "Checking cluster capacity...")
	err = cli.K8sClient.CheckNodeCapacity()
	if err != nil {
		return err
	}

	// Get available nodeport
	port, err := cli.K8sClient.AllocateNodePort()
	if err != nil {
		return fmt.Errorf("cannot allocate node port: %w", err)
	}

	// Create Minecraft server
	fmt.Fprintf(os.Stderr, "Creating server %s...\n", input.ServerName)
	spec := k8s.ServerSpec{Version: input.Version, GameMode: input.GameMode, MaxPlayers: input.MaxPlayers}
	err = cli.K8sClient.CreateServer(input.ServerName, cli.AppConfig.Username, port, selectedImage, spec)
	if err != nil {
		return fmt.Errorf("cannot create server: %w", err)
	}

	// Wait for pod to be ready
	fmt.Fprintln(os.Stderr, "Waiting for server to be ready...")
	err = cli.K8sClient.WaitForReady(input.ServerName)
	if err != nil {
		return fmt.Errorf("server %s unable to start: %w", input.ServerName, err)
	}

	fmt.Fprintf(os.Stderr, "Server %s is ready at %s:%d\n", input.ServerName, cli.AppConfig.ClusterIP, port)

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
	createCmd.Flags().StringVar(&serverImage, "server-image", "", "Minecraft server container image (default: "+config.ServerImage+")")
	serverCmd.AddCommand(createCmd)
}
