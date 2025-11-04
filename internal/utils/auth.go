package utils

import (
	"errors"
	"fmt"

	"github.com/fosrl/cli/internal/api"
	"github.com/spf13/viper"
)

// EnsureLoggedIn checks if the user is logged in by verifying:
// 1. A userId exists in the viper config
// 2. A session token exists in the key store
// Returns an error if the user is not logged in, nil otherwise.
func EnsureLoggedIn() error {
	// Check for userId in config
	userID := viper.GetString("userId")
	if userID == "" {
		return errors.New("No user ID found in config. Please login first")
	}

	// Check for session token in keyring
	_, err := api.GetSessionToken()
	if err != nil {
		return fmt.Errorf("No session found in key store. Please login first")
	}

	return nil
}
