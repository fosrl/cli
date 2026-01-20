package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/fingerprint"
)

// EnsureOlmCredentials ensures that OLM credentials exist and are valid.
// It checks if OLM credentials exist locally, verifies them on the server,
// and creates new ones if they don't exist or are invalid.
//
// If new ones are created, a "true" is returned to indicate we need to
// save the new credentials to disk.
func EnsureOlmCredentials(client *api.Client, account *config.Account) (bool, error) {
	userID := account.UserID

	if account.OlmCredentials != nil {
		serverCreds, err := client.GetUserOlm(userID, account.OlmCredentials.ID)
		if err == nil && serverCreds != nil {
			return false, nil
		}

		// If getting OLM fails, the OLM might not exist.
		// This requires regeneration; in case of any errors
		// that are not API-related, these are likely not
		// related to the credentials and should be bubbled up.
		if _, ok := err.(*api.ErrorResponse); !ok {
			return false, fmt.Errorf("failed to get OLM: %w", err)
		}

		// Clear invalid credentials so we can try to create new ones
		account.OlmCredentials = nil
	}

	// First, attempt to recover any credentials on any other machines
	// using the platform fingerprint.
	//
	// Use the cached one if it is available, since this is cached
	// by a privileged process that has access to fingerprinting
	// attributes like DMI information.
	configDir, _ := config.GetPangolinConfigDir()
	cachedPlatfromFingerprintFilename := filepath.Join(configDir, "platform_fingerprint")

	if cachedFingerprint, err := os.ReadFile(cachedPlatfromFingerprintFilename); err == nil {
		if recoveredOlm, err := client.RecoverOlmFromFingerprint(userID, string(cachedFingerprint)); err == nil {
			account.OlmCredentials = &config.OlmCredentials{
				ID:     recoveredOlm.OlmID,
				Secret: recoveredOlm.Secret,
			}

			return true, nil
		}
	}

	fp := fingerprint.GatherFingerprintInfo()

	if recoveredOlm, err := client.RecoverOlmFromFingerprint(userID, fp.PlatformFingerprint); err == nil {
		account.OlmCredentials = &config.OlmCredentials{
			ID:     recoveredOlm.OlmID,
			Secret: recoveredOlm.Secret,
		}

		return true, nil
	}

	newOlm, err := client.CreateOlm(userID, fingerprint.GetDeviceName())
	if err != nil {
		return false, fmt.Errorf("failed to create OLM: %w", err)
	}

	account.OlmCredentials = &config.OlmCredentials{
		ID:     newOlm.OlmID,
		Secret: newOlm.Secret,
	}

	return true, nil
}

// EnsureOrgAccess ensures that the user has access to the organization
func EnsureOrgAccess(client *api.Client, account *config.Account) error {
	// Get org via API to ensure it exists
	_, err := client.GetOrg(account.OrgID)
	if err != nil {
		return err
	}

	// Check org user access and policies
	accessResponse, err := client.CheckOrgUserAccess(account.OrgID, account.UserID)
	if err != nil {
		return err
	}

	// Check if user is allowed access
	if !accessResponse.Allowed {
		// Get hostname base URL for constructing the web URL
		url := fmt.Sprintf("%s/%s", FormatHostnameBaseURL(account.Host), account.OrgID)
		return fmt.Errorf("Organization policy is preventing you from connecting. Please visit %s to complete required steps", url)
	}

	return nil
}

// CheckBlockedBeforeConnect checks if the OLM is blocked before attempting to connect.
// This should only be called when the user attempts to connect, not during authentication.
// Returns an error if the account is blocked. If the check fails (network error, etc.),
// returns an error that the caller should log but allow the connection attempt to proceed
// (the server will reject if truly blocked).
func CheckBlockedBeforeConnect(client *api.Client, account *config.Account) error {
	if account.OlmCredentials == nil {
		// No OLM credentials, can't check blocked status
		return nil
	}

	userID := account.UserID
	olmID := account.OlmCredentials.ID
	var orgID string
	if account.OrgID != "" {
		orgID = account.OrgID
	}

	// Get OLM with optional orgId parameter
	var olm *api.Olm
	var err error
	if orgID != "" {
		olm, err = client.GetUserOlm(userID, olmID, orgID)
	} else {
		olm, err = client.GetUserOlm(userID, olmID)
	}

	if err != nil {
		// If check fails (network error, etc.), log but allow connection attempt
		// The server will reject if truly blocked
		return fmt.Errorf("failed to check blocked status: %w", err)
	}

	// Check if blocked
	if olm != nil && olm.Blocked != nil && *olm.Blocked {
		return fmt.Errorf("Your device is blocked in this organization. Contact your admin for more information.")
	}

	return nil
}

// UserDisplayName returns a display name for a user with precedence:
// email > name > username > "User"
func UserDisplayName(user *api.User) string {
	if user.Email != "" {
		return user.Email
	}
	if user.Name != nil && *user.Name != "" {
		return *user.Name
	}
	if user.Username != nil && *user.Username != "" {
		return *user.Username
	}
	return "User"
}

// AccountDisplayName returns a display name for an account with precedence:
// email > name > username > "Account"
func AccountDisplayName(account *config.Account) string {
	if account.Email != "" {
		return account.Email
	}
	if account.Name != nil && *account.Name != "" {
		return *account.Name
	}
	if account.Username != nil && *account.Username != "" {
		return *account.Username
	}
	return "Account"
}

// AccountDisplayNameWithHost returns a display name for an account with hostname suffix
// when multiple accounts might share the same email. Format: "displayName @ hostname"
func AccountDisplayNameWithHost(account *config.Account) string {
	displayName := AccountDisplayName(account)
	if account.Host != "" {
		return fmt.Sprintf("%s @ %s", displayName, account.Host)
	}
	return displayName
}
