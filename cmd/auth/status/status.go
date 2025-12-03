package status

import (
	"fmt"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  "Check if you are logged in and view your account information",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if user info exists in config
		userID := viper.GetString("userId")
		email := viper.GetString("email")

		// If no user info in config, user is not logged in (never logged in)
		if userID == "" && email == "" {
			utils.Info("Status: Not logged in")
			utils.Info("Run 'pangolin login' to authenticate")
			return
		}

		// User info exists in config, try to get user from API
		user, err := api.GlobalClient.GetUser()
		if err != nil {
			// Unable to get user - consider logged out (previously logged in but now not)
			utils.Info("Status: Logged out")
			utils.Info("Your session has expired or is invalid")
			utils.Info("Run 'pangolin login' to authenticate again")
			return
		}

		// Successfully got user - logged in
		utils.Success("Status: Logged in")
		// Show hostname if available
		hostname := viper.GetString("hostname")
		if hostname != "" {
			utils.Info("@ %s", hostname)
		}
		fmt.Println()

		// Display user information
		displayName := user.Email
		if user.Username != nil && *user.Username != "" {
			displayName = *user.Username
		} else if user.Name != nil && *user.Name != "" {
			displayName = *user.Name
		}

		if displayName != "" {
			utils.Info("User: %s", displayName)
		}
		if user.UserID != "" {
			utils.Info("User ID: %s", user.UserID)
		}
		fmt.Println()

		// Display organization information
		orgID := viper.GetString("orgId")
		if orgID != "" {
			utils.Info("Org ID: %s", orgID)
		}
	},
}
