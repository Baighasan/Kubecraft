package server

import (
	"github.com/baighasan/kubecraft/internal/cli"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage minecraft servers",
	Long:  "I'll think of this later",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	cli.RootCmd.AddCommand(serverCmd)
}
