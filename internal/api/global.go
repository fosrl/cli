package api

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "Pangolin: pangolin-cli"
	keyringUser    = "session-token"
	keyringOlmUser = "olm-credentials"
)

var GlobalClient *Client

// InitGlobalClient initializes the global API client with stored credentials.
// This function always succeeds in creating a client, even if no token is available.
// The client will be created without authentication if no token is found.
func InitGlobalClient() error {
	// Get hostname from viper config
	hostname := viper.GetString("hostname")
	if hostname == "" {
		hostname = "app.pangolin.net"
	}

	// Get session token from keyring (ignore errors - just use empty token if not found)
	token, _ := keyring.Get(keyringService, keyringUser)

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

// SaveSessionToken saves the session token to the OS keyring
func SaveSessionToken(token string) error {
	return keyring.Set(keyringService, keyringUser, token)
}

// GetSessionToken retrieves the session token from the OS keyring
func GetSessionToken() (string, error) {
	return keyring.Get(keyringService, keyringUser)
}

// DeleteSessionToken deletes the session token from the OS keyring
func DeleteSessionToken() error {
	return keyring.Delete(keyringService, keyringUser)
}

// SaveOlmCredentials saves OLM credentials (olmId.secret) to the OS keyring
// The userId is used as part of the keyring key to allow multiple users on the same machine
func SaveOlmCredentials(userID, olmID, secret string) error {
	if userID == "" {
		return fmt.Errorf("userId is required to save OLM credentials")
	}
	credentials := olmID + "." + secret
	keyringKey := keyringOlmUser + "$" + userID
	return keyring.Set(keyringService, keyringKey, credentials)
}

// GetOlmCredentials retrieves OLM credentials from the OS keyring
// The userId is used as part of the keyring key to allow multiple users on the same machine
// Returns olmID and secret, or an error if not found
func GetOlmCredentials(userID string) (string, string, error) {
	if userID == "" {
		return "", "", fmt.Errorf("userId is required to get OLM credentials")
	}
	keyringKey := keyringOlmUser + "$" + userID
	credentials, err := keyring.Get(keyringService, keyringKey)
	if err != nil {
		return "", "", err
	}

	// Split on the first dot (in case secret contains dots)
	parts := strings.SplitN(credentials, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid credentials format")
	}

	return parts[0], parts[1], nil
}

// DeleteOlmCredentials deletes OLM credentials from the OS keyring
// The userId is used as part of the keyring key to allow multiple users on the same machine
func DeleteOlmCredentials(userID string) error {
	if userID == "" {
		return fmt.Errorf("userId is required to delete OLM credentials")
	}
	keyringKey := keyringOlmUser + "$" + userID
	return keyring.Delete(keyringService, keyringKey)
}
