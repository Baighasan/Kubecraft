package server

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/baighasan/kubecraft/internal/cli"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list", // todo: add feature to checking only running or only stopped servers
	Short: "List all Minecraft server",
	Long:  "I'll think of this later",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeList()
	},
}

func executeList() error {
	serverList, err := cli.K8sClient.ListServers()
	if err != nil {
		return fmt.Errorf("couldn't list servers: %w", err)
	}

	if len(serverList) == 0 {
		fmt.Fprintln(os.Stderr, "No servers found")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "NAME\tSTATUS\tPORT\tAGE\n")
		for _, s := range serverList {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", s.Name, s.Status, s.NodePort, formatAge(s.Age))
		}
		w.Flush()
	}

	return nil
}

func formatAge(created time.Time) string {
	d := time.Since(created)
	if d.Hours() >= 24 {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

func init() {
	serverCmd.AddCommand(listCmd)
}
