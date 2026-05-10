package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// ErrClusterNotInitialized indicates cluster ip is missing from config
var ErrClusterNotInitialized = errors.New("cluster not initialized")

// Config represents the user's saved configuration
type Config struct {
	Username    string `yaml:"username"`
	Token       string `yaml:"token"`
	ClusterIP   string `yaml:"clusterIP"`
	TLSInsecure bool   `yaml:"tlsInsecure"`
}

// GetConfigPath returns the path to ~/.kubecraft/config
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".kubecraft/config")

	return configPath, nil
}

// CheckConfigExists checks if the config file exists
func CheckConfigExists() (bool, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return false, err
	}

	_, err = os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if config exists: %w", err)
	}

	return true, nil
}

// SaveConfig writes the config to disk
func SaveConfig(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("getting config path: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(configPath), 0o755)
	if err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	marshal, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config to yaml: %w", err)
	}

	err = os.WriteFile(configPath, marshal, 0o600)
	if err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// LoadConfig reads the config from disk
func LoadConfig() (*Config, error) {
	configExists, err := CheckConfigExists()
	if err != nil {
		return nil, fmt.Errorf("checking if config exists: %w", err)
	}
	if !configExists {
		return nil, fmt.Errorf("config file does not exist")
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("getting config path: %w", err)
	}

	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	config := &Config{}
	err = yaml.Unmarshal(file, config)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return config, nil
}

// ValidateForRegister checks that cluster ip exists before attempting to register
func (c *Config) ValidateForRegister() error {
	if err := c.validateClusterInitialized(); err != nil {
		return fmt.Errorf("validating cluster initialization: %w", err)
	}

	return nil
}

// ValidateForServer checks that the config has all the required fields before running server commands
func (c *Config) ValidateForServer() error {
	if err := c.validateClusterInitialized(); err != nil {
		return fmt.Errorf("validating cluster initialization: %w", err)
	}

	if err := c.validateAuth(); err != nil {
		return fmt.Errorf("validating auth creds: %w", err)
	}

	return nil
}

// validateClusterInitialized checks that cluster ip exists in config
func (c *Config) validateClusterInitialized() error {
	if c.ClusterIP == "" {
		return ErrClusterNotInitialized
	}

	return nil
}

// validateAuth checks that credentials for auth are present in config
func (c *Config) validateAuth() error {
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.Token == "" {
		return fmt.Errorf("token is missing")
	}
	return nil
}

// APIEndpoint constructs endpoint to hit cluster api
func (c *Config) APIEndpoint() (string, error) {
	if err := c.validateClusterInitialized(); err != nil {
		return "", fmt.Errorf("failed to construct endpoint: %w", err)
	}

	hostPort := net.JoinHostPort(c.ClusterIP, strconv.Itoa(ClusterAPIPort))
	apiURL := "https://" + hostPort

	return apiURL, nil
}

// RegistrationEndpoint constructs endpoint to hit registration service
func (c *Config) RegistrationEndpoint() (string, error) {
	if err := c.validateClusterInitialized(); err != nil {
		return "", fmt.Errorf("failed to construct endpoint: %w", err)
	}

	regHostPort := net.JoinHostPort(c.ClusterIP, strconv.Itoa(RegistrationServicePort))
	regURL := "http://" + regHostPort

	return regURL, nil
}
