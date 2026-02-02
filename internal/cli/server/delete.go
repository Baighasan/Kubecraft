package server

import (
	"bufio"
	"fmt"
	"os"

	"github.com/baighasan/kubecraft/internal/cli"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <server-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Delete a Minecraft server",
	Long:  "I'll think of this later",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		return executeDelete(serverName)
	},
}

func executeDelete(serverName string) error {
	// Verify server exists
	serverExists, err := cli.K8sClient.ServerExists(serverName)
	if err != nil {
		return fmt.Errorf("could not check if server exists: %v", err)
	}
	if !serverExists {
		return fmt.Errorf("server %s does not exist", serverName)
	}

	// Prompt user to type server name to confirm
	var input string
	fmt.Fprintf(os.Stderr, "Enter %s to confirm", serverName)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input = scanner.Text()
	}
	if input != serverName {
		fmt.Fprintf(os.Stderr, "Server name does not match, cancelling")
		return nil
	}

	// Delete the server
	err = cli.K8sClient.DeleteServer(serverName)
	if err != nil {
		return fmt.Errorf("could not delete server: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Server %s successfully deleted, data permanently gone", serverName)
	return nil
}

func init() {
	serverCmd.AddCommand(deleteCmd)
}
