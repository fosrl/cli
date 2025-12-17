package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// GetPangolinConfigDir returns the path to the .pangolin directory and ensures it exists
func GetPangolinConfigDir() (string, error) {
	homeDir, err := GetOriginalUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	pangolinDir := filepath.Join(homeDir, ".config", "pangolin")
	if err := os.MkdirAll(pangolinDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", pangolinDir, err)
	}

	return pangolinDir, nil
}

// GetOriginalUserHomeDir returns the home directory of the original user
// (the user who invoked the command, not the effective user when running with sudo).
// This ensures that config files work both with and without sudo.
func GetOriginalUserHomeDir() (string, error) {
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
