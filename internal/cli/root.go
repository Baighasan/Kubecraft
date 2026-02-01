package cli

import (
	"fmt"
	"os"

	"github.com/baighasan/kubecraft/internal/config"
	"github.com/baighasan/kubecraft/internal/k8s"
	"github.com/spf13/cobra"
)

var (
	AppConfig *config.Config
	K8sClient *k8s.Client
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "kubecraft",
	Short: "Minecraft server management cli",
	Long:  "todo",
}

func init() {
	// Persistent flags available to all subcommands
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Check config exists, load it, and create client (register command doesn't need config)
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "register" {
			return nil
		}

		configExists, err := config.CheckConfigExists()
		if err != nil {
			return fmt.Errorf("error while checking config exists: %v", err)
		}
		if !configExists {
			return fmt.Errorf("please register first by running kubecraft register")
		}

		AppConfig, err = config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error while loading config: %v", err)
		}

		K8sClient, err = k8s.NewClientFromToken(AppConfig.Token, config.ClusterEndpoint)
		if err != nil {
			return fmt.Errorf("error while creating k8s client: %v", err)
		}

		return nil
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Oops. An error while executing Kubecraft '%s'\n", err)
		os.Exit(1)
	}
}
