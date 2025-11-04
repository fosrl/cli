package logout

import (
	"time"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from Pangolin",
	Long:  "Logout and clear your session",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if client is running before logout
		olmClient := olm.NewClient("")
		if olmClient.IsRunning() {
			// Prompt user to confirm they want to disconnect the client
			var confirm bool
			confirmForm := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("A client is currently running. Logging out will disconnect it.").
						Description("Do you want to continue?").
						Value(&confirm),
				),
			)

			if err := confirmForm.Run(); err != nil {
				utils.Error("Error: %v", err)
				return
			}

			if !confirm {
				utils.Info("Logout cancelled")
				return
			}

			// Kill the client without showing TUI
			_, err := olmClient.Exit()
			if err != nil {
				utils.Warning("Failed to send exit signal to client: %v", err)
			} else {
				// Wait for client to stop (poll until socket is gone)
				maxWait := 10 * time.Second
				pollInterval := 200 * time.Millisecond
				elapsed := time.Duration(0)
				for olmClient.IsRunning() && elapsed < maxWait {
					time.Sleep(pollInterval)
					elapsed += pollInterval
				}
				if olmClient.IsRunning() {
					utils.Warning("Client did not stop within timeout")
				}
			}
		}

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
