package server

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
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
	return runCreateFromPrompts(os.Stdin, os.Stderr, executeCreateWithInput)
}

func runCreateFromPrompts(reader io.Reader, writer io.Writer, execute func(createInput) error) error {
	scanner := bufio.NewScanner(reader)

	input, err := promptCreateInputWithScanner(scanner, writer)
	if err != nil {
		return err
	}

	confirmed, err := promptConfirmationWithScanner(scanner, writer, input)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(writer, "Server creation cancelled")
		return nil
	}

	return execute(input)
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

	if err := validateCreateInput(input); err != nil {
		return fmt.Errorf("invalid create input: %w", err)
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

func validateCreateInput(input createInput) error {
	if err := ValidateServerName(input.ServerName); err != nil {
		return fmt.Errorf("invalid server name: %w", err)
	}

	if !slices.Contains(config.AllowedMinecraftVersions, input.Version) {
		return fmt.Errorf("version must be one of %v", config.AllowedMinecraftVersions)
	}

	if !slices.Contains(config.AllowedGameModes, input.GameMode) {
		return fmt.Errorf("game mode must be one of %v", config.AllowedGameModes)
	}

	if input.MaxPlayers < config.MinMaxPlayers || input.MaxPlayers > config.MaxMaxPlayers {
		return fmt.Errorf("max players must be between %d and %d", config.MinMaxPlayers, config.MaxMaxPlayers)
	}

	return nil
}

func promptCreateInput(reader io.Reader, writer io.Writer) (createInput, error) {
	scanner := bufio.NewScanner(reader)
	return promptCreateInputWithScanner(scanner, writer)
}

func promptCreateInputWithScanner(scanner *bufio.Scanner, writer io.Writer) (createInput, error) {
	version, err := promptMenu(scanner, writer, "Select Minecraft version", config.AllowedMinecraftVersions)
	if err != nil {
		return createInput{}, err
	}

	serverName, err := promptServerName(scanner, writer)
	if err != nil {
		return createInput{}, err
	}

	gameMode, err := promptMenu(scanner, writer, "Select game mode", config.AllowedGameModes)
	if err != nil {
		return createInput{}, err
	}

	maxPlayers, err := promptMaxPlayers(scanner, writer)
	if err != nil {
		return createInput{}, err
	}

	return createInput{
		ServerName: serverName,
		Version:    version,
		GameMode:   gameMode,
		MaxPlayers: maxPlayers,
	}, nil
}

func promptMenu(scanner *bufio.Scanner, writer io.Writer, label string, options []string) (string, error) {
	for {
		fmt.Fprintf(writer, "%s:\n", label)
		for i, option := range options {
			fmt.Fprintf(writer, "  %d) %s\n", i+1, option)
		}
		fmt.Fprint(writer, "Enter choice number: ")

		input, err := scanTrimmedLine(scanner)
		if err != nil {
			return "", err
		}

		index, err := strconv.Atoi(input)
		if input == "" || err != nil || index < 1 || index > len(options) {
			fmt.Fprintln(writer, "Invalid selection, please choose a valid menu number")
			continue
		}

		return options[index-1], nil
	}
}

func promptServerName(scanner *bufio.Scanner, writer io.Writer) (string, error) {
	for {
		fmt.Fprint(writer, "Enter server name: ")
		name, err := scanTrimmedLine(scanner)
		if err != nil {
			return "", err
		}
		if name == "" {
			fmt.Fprintln(writer, "Invalid server name: input cannot be empty")
			continue
		}

		if err := ValidateServerName(name); err != nil {
			fmt.Fprintf(writer, "Invalid server name: %v\n", err)
			continue
		}

		return name, nil
	}
}

func promptMaxPlayers(scanner *bufio.Scanner, writer io.Writer) (int, error) {
	for {
		fmt.Fprintf(writer, "Select max players (%d-%d): ", config.MinMaxPlayers, config.MaxMaxPlayers)
		input, err := scanTrimmedLine(scanner)
		if err != nil {
			return 0, err
		}
		if input == "" {
			fmt.Fprintf(writer, "Invalid max players, enter a number between %d and %d\n", config.MinMaxPlayers, config.MaxMaxPlayers)
			continue
		}

		value, err := strconv.Atoi(input)
		if err != nil || value < config.MinMaxPlayers || value > config.MaxMaxPlayers {
			fmt.Fprintf(writer, "Invalid max players, enter a number between %d and %d\n", config.MinMaxPlayers, config.MaxMaxPlayers)
			continue
		}

		return value, nil
	}
}

func promptConfirmation(reader io.Reader, writer io.Writer, input createInput) (bool, error) {
	scanner := bufio.NewScanner(reader)
	return promptConfirmationWithScanner(scanner, writer, input)
}

func promptConfirmationWithScanner(scanner *bufio.Scanner, writer io.Writer, input createInput) (bool, error) {
	fmt.Fprintln(writer, "\nServer configuration:")
	fmt.Fprintf(writer, "  Version: %s\n", input.Version)
	fmt.Fprintf(writer, "  Server Name: %s\n", input.ServerName)
	fmt.Fprintf(writer, "  Game Mode: %s\n", input.GameMode)
	fmt.Fprintf(writer, "  Max Players: %d\n", input.MaxPlayers)
	fmt.Fprint(writer, "Proceed? (y/N): ")

	answer, err := scanTrimmedLine(scanner)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"), nil
}

func scanTrimmedLine(scanner *bufio.Scanner) (string, error) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		return "", fmt.Errorf("no input received")
	}

	return strings.TrimSpace(scanner.Text()), nil
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
