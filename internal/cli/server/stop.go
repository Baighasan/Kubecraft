package server

import (
	"fmt"
	"os"

	"github.com/baighasan/kubecraft/internal/cli"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <server-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Stop a Minecraft server",
	Long:  "I'll think of this later",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		return executeStop(serverName)
	},
}

func executeStop(serverName string) error {
	// Verify server exists
	serverExists, err := cli.K8sClient.ServerExists(serverName)
	if err != nil {
		return fmt.Errorf("could not check server existence: %w", err)
	}
	if !serverExists {
		return fmt.Errorf("server (%s) does not exist", serverName)
	}

	// Scale down server (statefulset)
	err = cli.K8sClient.ScaleServer(serverName, 0)
	if err != nil {
		return fmt.Errorf("could not stop server: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Server %s successfully stopped", serverName)
	return nil
}

func init() {
	serverCmd.AddCommand(stopCmd)
}
