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

	LogLevel             logger.LogLevel      `mapstructure:"log_level" json:"log_level"`
	LogFile              string               `mapstructure:"log_file" json:"log_file"`
	DisableUpdateCheck   bool                 `mapstructure:"disable_update_check" json:"disable_update_check"`
	DisableCompanionMode bool                 `mapstructure:"disable_companion_mode" json:"disable_companion_mode"`
	CompanionAppDataDirs CompanionAppDataDirs `mapstructure:"companion_app_data_dirs" json:"companion_app_data_dirs"`
	Up                   UpConfig             `mapstructure:"up" json:"up,omitempty"`
}

// UpConfig holds persistent defaults for pangolin up DNS-related flags.
// Pointer bools distinguish unset from explicitly false.
type UpConfig struct {
	TunnelDNS   *bool    `mapstructure:"tunnel_dns" json:"tunnel_dns,omitempty"`
	UpstreamDNS []string `mapstructure:"upstream_dns" json:"upstream_dns,omitempty"`
	OverrideDNS *bool    `mapstructure:"override_dns" json:"override_dns,omitempty"`
}

// CompanionAppDataDirs holds per-platform overrides for the desktop app data directory.
type CompanionAppDataDirs struct {
	Windows string `mapstructure:"windows" json:"windows,omitempty"`
	Darwin  string `mapstructure:"darwin" json:"darwin,omitempty"`
}

// ConfigOptions is the registry of keys supported by pangolin config get/set/list.
var ConfigOptions = []string{
	"log_level",
	"disable_update_check",
	"disable_companion_mode",
	"up.tunnel_dns",
	"up.upstream_dns",
	"up.override_dns",
}

// SupportedConfigKeys returns the settable config keys.
func SupportedConfigKeys() []string {
	out := make([]string, len(ConfigOptions))
	copy(out, ConfigOptions)
	return out
}

// CompanionAppDataDirForPlatform returns the configured override for the current OS.
func (c *Config) CompanionAppDataDirForPlatform() string {
	return companionAppDataDirForGOOS(c, runtime.GOOS)
}

func companionAppDataDirForGOOS(c *Config, goos string) string {
	switch goos {
	case "windows":
		return c.CompanionAppDataDirs.Windows
	case "darwin":
		return c.CompanionAppDataDirs.Darwin
	default:
		return ""
	}
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
	v.SetDefault("disable_companion_mode", false)
	v.SetDefault("companion_app_data_dirs", map[string]string{})

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

// CompanionModeEnabled reports whether companion mode is enabled in config.
func (c *Config) CompanionModeEnabled() bool {
	return !c.DisableCompanionMode
}

// SetCompanionModeEnabled updates the companion mode config flag.
func (c *Config) SetCompanionModeEnabled(enabled bool) {
	c.DisableCompanionMode = !enabled
}

// IsSet reports whether key was set via config file or environment variable.
// Keys that only have a built-in default are not considered set.
func (c *Config) IsSet(key string) bool {
	return c.v.IsSet(key)
}

// GetBool returns the boolean value for key from the merged config sources.
func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

// GetString returns the string value for key from the merged config sources.
func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

// GetStringSlice returns the string slice value for key from the merged config sources.
func (c *Config) GetStringSlice(key string) []string {
	return c.v.GetStringSlice(key)
}

// ConfigFilePath returns the path to the CLI config file.
func ConfigFilePath() (string, error) {
	dir, err := GetPangolinConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// SetKey sets a supported configuration key and syncs the in-memory struct.
func (c *Config) SetKey(key, value string) error {
	switch key {
	case "log_level":
		level := logger.LogLevel(value)
		switch level {
		case logger.LogLevelDebug, logger.LogLevelInfo:
			c.LogLevel = level
			c.v.Set(key, value)
		default:
			return fmt.Errorf("invalid log_level %q: must be debug or info", value)
		}
	case "disable_update_check":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		c.DisableUpdateCheck = b
		c.v.Set(key, b)
	case "disable_companion_mode":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		c.DisableCompanionMode = b
		c.v.Set(key, b)
	case "up.tunnel_dns":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		c.Up.TunnelDNS = &b
		c.v.Set(key, b)
	case "up.override_dns":
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		c.Up.OverrideDNS = &b
		c.v.Set(key, b)
	case "up.upstream_dns":
		servers := splitCommaList(value)
		c.Up.UpstreamDNS = servers
		c.v.Set(key, servers)
	default:
		return fmt.Errorf("unknown config key %q; supported keys: %s", key, strings.Join(SupportedConfigKeys(), ", "))
	}
	return nil
}

// GetKey returns a string representation of a supported configuration key.
func (c *Config) GetKey(key string) (string, error) {
	switch key {
	case "log_level":
		return string(c.LogLevel), nil
	case "log_file":
		return c.LogFile, nil
	case "disable_update_check":
		return fmt.Sprintf("%t", c.DisableUpdateCheck), nil
	case "disable_companion_mode":
		return fmt.Sprintf("%t", c.DisableCompanionMode), nil
	case "up.tunnel_dns":
		if !c.IsSet(key) {
			return "", errConfigKeyUnset(key)
		}
		return fmt.Sprintf("%t", c.GetBool(key)), nil
	case "up.override_dns":
		if !c.IsSet(key) {
			return "", errConfigKeyUnset(key)
		}
		return fmt.Sprintf("%t", c.GetBool(key)), nil
	case "up.upstream_dns":
		if !c.IsSet(key) {
			return "", errConfigKeyUnset(key)
		}
		return strings.Join(c.GetStringSlice(key), ","), nil
	default:
		return "", fmt.Errorf("unknown config key %q; supported keys: %s", key, strings.Join(SupportedConfigKeys(), ", "))
	}
}

func errConfigKeyUnset(key string) error {
	return fmt.Errorf("config key %q is not set", key)
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q: use true or false", value)
	}
}

func splitCommaList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *Config) Save() error {
	c.v.Set("log_level", c.LogLevel)
	c.v.Set("log_file", c.LogFile)
	c.v.Set("disable_update_check", c.DisableUpdateCheck)
	c.v.Set("disable_companion_mode", c.DisableCompanionMode)
	c.v.Set("companion_app_data_dirs", c.CompanionAppDataDirs)

	// Only persist up keys that were explicitly set so we do not write
	// zero-value bools that would later look like intentional overrides.
	if c.Up.TunnelDNS != nil {
		c.v.Set("up.tunnel_dns", *c.Up.TunnelDNS)
	}
	if c.Up.OverrideDNS != nil {
		c.v.Set("up.override_dns", *c.Up.OverrideDNS)
	}
	if c.Up.UpstreamDNS != nil {
		c.v.Set("up.upstream_dns", c.Up.UpstreamDNS)
	}

	dir, err := GetPangolinConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := c.v.ConfigFileUsed()
	if configFile == "" {
		configFile, err = ConfigFilePath()
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(configFile); errors.Is(err, os.ErrNotExist) {
		return c.v.WriteConfigAs(configFile)
	}

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
