package api

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "Pangolin: pangolin-cli"
	keyringUser    = "session-token"
)

var GlobalClient *Client

// getOriginalUserHomeDir returns the home directory of the original user
// (the user who invoked the command, not the effective user when running with sudo).
// This ensures that config files and keyring access work both with and without sudo.
func getOriginalUserHomeDir() (string, error) {
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

// withOriginalUserHome executes a function with HOME set to the original user's home directory.
// This ensures keyring access works both with and without sudo.
func withOriginalUserHome(fn func() error) error {
	// Get original user's home directory
	originalHome, err := getOriginalUserHomeDir()
	if err != nil {
		// If we can't get original user's home, try with current HOME
		return fn()
	}

	// Save current HOME
	currentHome := os.Getenv("HOME")

	// Set HOME to original user's home directory
	os.Setenv("HOME", originalHome)
	defer func() {
		// Restore original HOME
		if currentHome != "" {
			os.Setenv("HOME", currentHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	return fn()
}

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
	var token string
	withOriginalUserHome(func() error {
		token, _ = keyring.Get(keyringService, keyringUser)
		return nil
	})

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
	var err error
	withOriginalUserHome(func() error {
		err = keyring.Set(keyringService, keyringUser, token)
		return err
	})
	return err
}

// GetSessionToken retrieves the session token from the OS keyring
func GetSessionToken() (string, error) {
	var token string
	var err error
	withOriginalUserHome(func() error {
		token, err = keyring.Get(keyringService, keyringUser)
		return err
	})
	return token, err
}

// DeleteSessionToken deletes the session token from the OS keyring
func DeleteSessionToken() error {
	var err error
	withOriginalUserHome(func() error {
		err = keyring.Delete(keyringService, keyringUser)
		return err
	})
	return err
}

// SaveOlmCredentials saves OLM credentials to the OS keyring
// The userId is used as part of the keyring key to allow multiple users on the same machine
func SaveOlmCredentials(userID, olmID, secret string) error {
	if userID == "" {
		return fmt.Errorf("userId is required to save OLM credentials")
	}
	idKey := fmt.Sprintf("olm-id-%s", userID)
	secretKey := fmt.Sprintf("olm-secret-%s", userID)
	var err error
	withOriginalUserHome(func() error {
		if err = keyring.Set(keyringService, idKey, olmID); err != nil {
			return err
		}
		err = keyring.Set(keyringService, secretKey, secret)
		return err
	})
	return err
}

// GetOlmCredentials retrieves OLM credentials from the OS keyring
// The userId is used as part of the keyring key to allow multiple users on the same machine
// Returns olmID and secret, or an error if not found
func GetOlmCredentials(userID string) (string, string, error) {
	if userID == "" {
		return "", "", fmt.Errorf("userId is required to get OLM credentials")
	}
	idKey := fmt.Sprintf("olm-id-%s", userID)
	secretKey := fmt.Sprintf("olm-secret-%s", userID)
	var olmID, secret string
	var err error
	withOriginalUserHome(func() error {
		olmID, err = keyring.Get(keyringService, idKey)
		if err != nil {
			return err
		}
		secret, err = keyring.Get(keyringService, secretKey)
		return err
	})
	if err != nil {
		return "", "", err
	}

	return olmID, secret, nil
}

// DeleteOlmCredentials deletes OLM credentials from the OS keyring
// The userId is used as part of the keyring key to allow multiple users on the same machine
func DeleteOlmCredentials(userID string) error {
	if userID == "" {
		return fmt.Errorf("userId is required to delete OLM credentials")
	}
	idKey := fmt.Sprintf("olm-id-%s", userID)
	secretKey := fmt.Sprintf("olm-secret-%s", userID)
	var err error
	withOriginalUserHome(func() error {
		// Try to delete both entries, continue even if one doesn't exist
		if delErr := keyring.Delete(keyringService, idKey); delErr != nil && err == nil {
			err = delErr
		}
		if delErr := keyring.Delete(keyringService, secretKey); delErr != nil && err == nil {
			err = delErr
		}
		return err
	})
	return err
}
