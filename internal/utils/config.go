package utils

import (
	"strings"

	"github.com/spf13/viper"
)

const defaultHostname = "app.pangolin.net"

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
