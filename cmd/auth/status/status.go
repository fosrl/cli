package status

import (
	"fmt"

	"github.com/fosrl/cli/internal/accounts"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long:  "Check if you are logged in and view your account information",
	Run: func(cmd *cobra.Command, args []string) {
		apiClient := api.FromContext(cmd.Context())
		accountStore := accounts.FromContext(cmd.Context())

		account, err := accountStore.ActiveAccount()
		if err != nil {
			utils.Info("Status: %s", err)
			utils.Info("Run 'pangolin login' to authenticate")
			return
		}

		// User info exists in config, try to get user from API
		user, err := apiClient.GetUser()
		if err != nil {
			// Unable to get user - consider logged out (previously logged in but now not)
			utils.Info("Status: Logged out: %v", err)
			utils.Info("Your session has expired or is invalid")
			utils.Info("Run 'pangolin login' to authenticate again")
			return
		}

		// Successfully got user - logged in
		utils.Success("Status: Logged in")
		// Show hostname if available
		utils.Info("@ %s", account.Host)
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

		// Display organization information
		utils.Info("Org ID: %s", account.OrgID)
	},
}
