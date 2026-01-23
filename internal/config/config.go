package config

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fosrl/cli/internal/logger"
	"github.com/spf13/viper"
)

type Config struct {
	// All operations must happen to the configuration file,
	// so they must operate on separate Viper instances.
	v *viper.Viper

	LogLevel           logger.LogLevel `mapstructure:"log_level" json:"log_level"`
	LogFile            string          `mapstructure:"log_file" json:"log_file"`
	DisableUpdateCheck bool            `mapstructure:"disable_update_check" json:"disable_update_check"`
}

func newConfigViper() (*viper.Viper, error) {
	v := viper.New()

	dir, err := GetPangolinConfigDir()
	if err != nil {
		return nil, err
	}

	// Bind to environment variables of the same name
	v.SetEnvPrefix("PANGOLIN_CLI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	configFile := filepath.Join(dir, "config.json")
	v.SetConfigFile(configFile)
	v.SetConfigType("json")

	defaultLogPath := defaultLogPath()

	// Defaults
	v.SetDefault("log_level", "info")
	v.SetDefault("log_file", defaultLogPath)
	v.SetDefault("disable_update_check", false)

	return v, nil
}

func LoadConfig() (*Config, error) {
	v, err := newConfigViper()
	if err != nil {
		return nil, err
	}

	cfg := Config{v: v}

	if err := v.ReadInConfig(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := v.Unmarshal(&cfg); err != nil {
				return nil, err
			}

			return &cfg, nil
		}

		return nil, err
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	switch c.LogLevel {
	case logger.LogLevelDebug, logger.LogLevelInfo:
		return nil
	default:
		return fmt.Errorf("invalid log level: %v", c.LogLevel)
	}
}

func (c *Config) Save() error {
	c.v.Set("log_level", c.LogLevel)
	c.v.Set("log_file", c.LogFile)
	c.v.Set("disable_update_check", c.DisableUpdateCheck)

	return c.v.WriteConfig()
}

// GetPangolinConfigDir returns the path to the .pangolin directory and ensures it exists
func GetPangolinConfigDir() (string, error) {
	homeDir, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	pangolinDir := filepath.Join(homeDir, ".config", "pangolin")

	return pangolinDir, nil
}

// userHomeDir returns the home directory of the original user
// (the user who invoked the command, not the effective user when running with sudo).
// This ensures that config files work both with and without sudo.
func userHomeDir() (string, error) {
	// Check if we're running under sudo - SUDO_USER contains the original user
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		// We're running with sudo, get the original user's home directory
		u, err := user.Lookup(sudoUser)
		if err != nil {
			return "", fmt.Errorf("failed to lookup original user %s: %w", sudoUser, err)
		}
		return u.HomeDir, nil
	}

	// Not running with sudo, use current user's home directory
	return os.UserHomeDir()
}

// defaultLogPath returns the default log file path for client logs
func defaultLogPath() string {
	pangolinDir, err := GetPangolinConfigDir()
	if err != nil {
		return "/tmp/olm.log"
	}

	logsDir := filepath.Join(pangolinDir, "logs")
	return filepath.Join(logsDir, "client.log")
}

// GetFingerprintDir returns the directory for storing the platform fingerprint.
// On Linux, this uses /etc/pangolin since the fingerprint is machine-specific
// and needs to be written by a privileged process but readable by all users.
// On other platforms, it falls back to the user config directory.
func GetFingerprintDir() (string, error) {
	// On Linux, prefer /etc/pangolin for system-wide fingerprint storage
	if runtime.GOOS == "linux" {
		return "/etc/pangolin", nil
	}

	// On other platforms, use the user config directory
	return GetPangolinConfigDir()
}

// GetFingerprintFilePath returns the full path to the platform fingerprint file.
func GetFingerprintFilePath() (string, error) {
	dir, err := GetFingerprintDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "platform_fingerprint"), nil
}
