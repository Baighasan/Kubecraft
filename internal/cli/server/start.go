package server

import (
	"fmt"
	"os"

	"github.com/baighasan/kubecraft/internal/cli"
	"github.com/baighasan/kubecraft/internal/config"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <server-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Start up a Minecraft server",
	Long:  "I'll think of this later",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		return executeStart(serverName)
	},
}

func executeStart(serverName string) error {
	// Validate server exists
	serverExists, err := cli.K8sClient.ServerExists(serverName)
	if err != nil {
		return fmt.Errorf("couldn't check server (%s) existence: %v", serverName, err)
	}
	if !serverExists {
		return fmt.Errorf("server (%s) does not exist", serverName)
	}

	// Scale up server (statefulset)
	err = cli.K8sClient.ScaleServer(serverName, 1)
	if err != nil {
		return fmt.Errorf("could not start server (%s): %v", serverName, err)
	}

	// Wait for server to become ready
	err = cli.K8sClient.WaitForReady(serverName)
	if err != nil {
		return fmt.Errorf("server %s unresponsive: %v", serverName, err)
	}

	// Get nodeport
	serverPort, err := cli.K8sClient.GetNodePort(serverName)
	if err != nil {
		return fmt.Errorf("couldn't get node port: %v", err)
	}

	// Print success message to user
	fmt.Fprintf(os.Stdout, "Server (%s) is ready at %s:%d.\n", serverName, config.ClusterEndpoint, serverPort)

	return nil
}

func init() {
	serverCmd.AddCommand(startCmd)
}
