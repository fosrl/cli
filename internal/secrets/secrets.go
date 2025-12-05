package secrets

import (
	"fmt"

	"github.com/spf13/viper"
)

// SaveSessionToken saves the session token to the config file
func SaveSessionToken(token string) error {
	viper.Set("sessionToken", token)
	return viper.WriteConfig()
}

// GetSessionToken retrieves the session token from the config file
func GetSessionToken() (string, error) {
	token := viper.GetString("sessionToken")
	if token == "" {
		return "", fmt.Errorf("session token not found")
	}
	return token, nil
}

// DeleteSessionToken deletes the session token from the config file
func DeleteSessionToken() error {
	viper.Set("sessionToken", "")
	return viper.WriteConfig()
}

// SaveOlmCredentials saves OLM credentials to the config file
func SaveOlmCredentials(userID, olmID, secret string) error {
	if userID == "" {
		return fmt.Errorf("userId is required to save OLM credentials")
	}
	viper.Set(fmt.Sprintf("olmCredentials.%s.id", userID), olmID)
	viper.Set(fmt.Sprintf("olmCredentials.%s.secret", userID), secret)
	return viper.WriteConfig()
}

// GetOlmCredentials retrieves OLM credentials from the config file
func GetOlmCredentials(userID string) (string, string, error) {
	if userID == "" {
		return "", "", fmt.Errorf("userId is required to get OLM credentials")
	}
	olmID := viper.GetString(fmt.Sprintf("olmCredentials.%s.id", userID))
	secret := viper.GetString(fmt.Sprintf("olmCredentials.%s.secret", userID))
	if olmID == "" || secret == "" {
		return "", "", fmt.Errorf("OLM credentials not found for user %s", userID)
	}
	return olmID, secret, nil
}

// DeleteOlmCredentials deletes OLM credentials from the config file
func DeleteOlmCredentials(userID string) error {
	if userID == "" {
		return fmt.Errorf("userId is required to delete OLM credentials")
	}
	viper.Set(fmt.Sprintf("olmCredentials.%s.id", userID), "")
	viper.Set(fmt.Sprintf("olmCredentials.%s.secret", userID), "")
	return viper.WriteConfig()
}
