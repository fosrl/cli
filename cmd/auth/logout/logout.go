package logout

import (
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Pangolin",
	Long:  "Logout and clear your session",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if there's an active session in the key store
		_, err := api.GetSessionToken()
		if err != nil {
			// No session found - user is already logged out
			utils.Success("Already logged out!")
			return
		}

		// Get user info before clearing config
		accountName := viper.GetString("email")
		if accountName == "" {
			// Try to get username from API as fallback
			if user, err := api.GlobalClient.GetUser(); err == nil {
				if user.Username != "" {
					accountName = user.Username
				} else if user.Email != "" {
					accountName = user.Email
				}
			}
		}

		// Try to logout from server (client is always initialized)
		if err := api.GlobalClient.Logout(); err != nil {
			// Ignore logout errors - we'll still clear local data
			utils.Debug("Failed to logout from server: %v", err)
		}

		// Clear session token from keyring
		if err := api.DeleteSessionToken(); err != nil {
			// Ignore error if token doesn't exist (already logged out)
			utils.Error("Failed to delete session token: %v", err)
			return
		}

		// Clear user-specific config values
		viper.Set("userId", "")
		viper.Set("email", "")
		viper.Set("orgId", "")

		if err := viper.WriteConfig(); err != nil {
			utils.Error("Failed to clear config: %v", err)
			return
		}

		// Re-initialize the global client without a token
		if err := api.InitGlobalClient(); err != nil {
			// This should never happen, but log it
			utils.Warning("Failed to re-initialize API client: %v", err)
		}

		// Print logout message with account name
		if accountName != "" {
			utils.Success("Logged out of Pangolin account %s", accountName)
		} else {
			utils.Success("Logged out of Pangolin account")
		}
	},
}
