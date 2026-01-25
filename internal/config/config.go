package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the user's saved configuration
type Config struct {
	Username        string `yaml:"username"`
	Token           string `yaml:"token"`
	ClusterEndpoint string `yaml:"cluster_endpoint"`
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

	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	marshal, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config to yaml: %w", err)
	}

	err = os.WriteFile(configPath, marshal, 0600)
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

	err = config.Validate()
	if err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return config, nil
}

// Validate checks that the config has all the required fields
func (c *Config) Validate() error {
	if len(c.Username) == 0 {
		return fmt.Errorf("username is required")
	}

	if len(c.Token) == 0 {
		return fmt.Errorf("token is missing")
	}

	if len(c.ClusterEndpoint) == 0 {
		return fmt.Errorf("cluster endpoint is missing")
	}

	return nil
}
