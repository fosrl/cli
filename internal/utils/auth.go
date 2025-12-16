package utils

import (
	"errors"
	"fmt"
	"os"
	"os/user"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/secrets"
	"github.com/spf13/viper"
)

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

// Returns an error if the user is not logged in, nil otherwise.
func EnsureLoggedIn() error {
	// Check for userId in config
	userID := viper.GetString("userId")
	if userID == "" {
		return errors.New("Please log in first. Run `pangolin login` to login")
	}

	// Check for session token in config
	_, err := secrets.GetSessionToken()
	if err != nil {
		return fmt.Errorf("Please log in first. Run `pangolin login` to login")
	}

	// Get user via API to ensure the user exists
	_, err = api.GlobalClient.GetUser()
	if err != nil {
		return fmt.Errorf("failed to get user information: %w", err)
	}

	return nil
}

// GetDeviceName returns a human-readable device name
func GetDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "Unknown Device"
	}
	return hostname
}

// EnsureOlmCredentials ensures that OLM credentials exist and are valid.
// It checks if OLM credentials exist locally, verifies them on the server,
// and creates new ones if they don't exist or are invalid.
func EnsureOlmCredentials(userID string) error {
	if userID == "" {
		return errors.New("userId is required")
	}

	// Check if OLM credentials already exist locally
	olmID, _, err := secrets.GetOlmCredentials(userID)
	if err == nil && olmID != "" {
		// Verify OLM exists on server by getting the OLM directly
		olm, err := api.GlobalClient.GetUserOlm(userID, olmID)
		if err == nil && olm != nil {
			// Verify the olmID matches
			if olm.OlmID == olmID {
				return nil
			} else {
				Error("OLM ID mismatch - olm olmID: %s, stored olmID: %s", olm.OlmID, olmID)
				// Clear invalid credentials
				secrets.DeleteOlmCredentials(userID)
			}
		} else {
			// If getting OLM fails, the OLM might not exist
			_, ok := err.(*api.ErrorResponse)
			if !ok {
				return fmt.Errorf("failed to get OLM: %w", err)
			}

			// Clear invalid credentials so we can try to create new ones
			secrets.DeleteOlmCredentials(userID)
		}
	}

	// If credentials don't exist or were cleared, create new ones
	_, _, err = secrets.GetOlmCredentials(userID)
	if err != nil {
		// Get friendly device name
		deviceName := GetDeviceName()

		olmResponse, err := api.GlobalClient.CreateOlm(userID, deviceName)
		if err != nil {
			return fmt.Errorf("failed to create OLM: %w", err)
		}

		// Save OLM credentials
		if err := secrets.SaveOlmCredentials(userID, olmResponse.OlmID, olmResponse.Secret); err != nil {
			return fmt.Errorf("failed to save OLM credentials: %w", err)
		}
	}

	return nil
}

// EnsureOrgAccess ensures that the user has access to the organization
func EnsureOrgAccess(orgID, userID string) error {
	if orgID == "" {
		return errors.New("orgId is required")
	}
	if userID == "" {
		return errors.New("userId is required")
	}

	// Get org via API to ensure it exists
	_, err := api.GlobalClient.GetOrg(orgID)
	if err != nil {
		return err
	}

	// Check org user access and policies
	accessResponse, err := api.GlobalClient.CheckOrgUserAccess(orgID, userID)
	if err != nil {
		return err
	}

	// Check if user is allowed access
	if !accessResponse.Allowed {
		// Get hostname base URL for constructing the web URL
		hostname := GetHostnameBaseURL()
		url := fmt.Sprintf("%s/%s", hostname, orgID)
		return fmt.Errorf("Organization policy is preventing you from connecting. Please visit %s to complete required steps", url)
	}

	return nil
}
