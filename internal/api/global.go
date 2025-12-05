package api

import (
	"fmt"
	"strings"

	"github.com/fosrl/cli/internal/secrets"
	"github.com/spf13/viper"
)

var GlobalClient *Client

// InitGlobalClient initializes the global API client with stored credentials.
// The client will be created without authentication if no token is found.
func InitGlobalClient() error {
	// Get hostname from viper config
	hostname := viper.GetString("hostname")
	if hostname == "" {
		hostname = "app.pangolin.net"
	}

	// Get session token from config (ignore errors - just use empty token if not found)
	token, _ := secrets.GetSessionToken()

	// Build base URL (hostname should already include protocol from login)
	baseURL := hostname
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		// If no protocol, default to https
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/") + "/api/v1"

	// Create API client (this should never fail, but handle it just in case)
	client, err := NewClient(ClientConfig{
		BaseURL:           baseURL,
		AgentName:         "pangolin-cli",
		Token:             token,
		SessionCookieName: "p_session_token",
		CSRFToken:         "x-csrf-protection",
	})
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	GlobalClient = client
	return nil
}
