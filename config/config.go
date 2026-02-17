package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName  = ".config/solo-cli"
	configFileName = "config.json"
)

// DefaultUserAgent is the default user agent string
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36"

// Config holds the user credentials for SOLO.ro
type Config struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	CompanyID string `json:"company_id"`
	PageSize  int    `json:"page_size"`
	UserAgent string `json:"user_agent"`
}

// ErrCredentialsMissing is returned when username or password is empty
var ErrCredentialsMissing = errors.New("credentials missing: please set username and password in config file")

// customConfigPath allows overriding the default config path
var customConfigPath string

// SetConfigPath sets a custom path for the config file
func SetConfigPath(path string) {
	customConfigPath = path
}

// GetConfigDir returns the full path to the config directory
func GetConfigDir() (string, error) {
	if customConfigPath != "" {
		return filepath.Dir(customConfigPath), nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configDirName), nil
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	if customConfigPath != "" {
		return customConfigPath, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, configDirName, configFileName), nil
}

// EnsureExists creates the config directory and an empty config file if they don't exist
func EnsureExists() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create config file with all parameters
		emptyConfig := Config{
			Username:  "",
			Password:  "",
			CompanyID: "",
			PageSize:  100,
			UserAgent: DefaultUserAgent,
		}
		data, err := json.MarshalIndent(emptyConfig, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(configPath, data, 0600)
	}

	return nil
}

// Load reads and parses the config file
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON in config file: %w\nPlease check the syntax at: %s", err, configPath)
	}

	// Validate credentials
	if cfg.Username == "" || cfg.Password == "" {
		return nil, ErrCredentialsMissing
	}

	return &cfg, nil
}
