package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const defaultHostname = "app.pangolin.net"

// GetPangolinDir returns the path to the .pangolin directory and ensures it exists
func GetPangolinDir() (string, error) {
	homeDir, err := GetOriginalUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	pangolinDir := filepath.Join(homeDir, ".pangolin")
	if err := os.MkdirAll(pangolinDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .pangolin directory: %w", err)
	}

	return pangolinDir, nil
}

// GetHostname returns the hostname from config with default fallback.
// It returns the hostname with protocol if present, or defaults to "app.pangolin.net".
func GetHostname() string {
	hostname := viper.GetString("hostname")
	if hostname == "" {
		return defaultHostname
	}
	return hostname
}

// GetHostnameBaseURL returns the hostname formatted as a base URL (with protocol, without /api/v1).
// This is useful for constructing URLs to the web interface.
func GetHostnameBaseURL() string {
	hostname := GetHostname()

	// Ensure hostname has protocol
	if !strings.HasPrefix(hostname, "http://") && !strings.HasPrefix(hostname, "https://") {
		hostname = "https://" + hostname
	}

	// Remove /api/v1 suffix if present
	hostname = strings.TrimSuffix(hostname, "/api/v1")
	hostname = strings.TrimSuffix(hostname, "/")

	return hostname
}
