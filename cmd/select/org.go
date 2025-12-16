package selectcmd

import (
	"fmt"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/tui"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var flagOrgID string

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Select an organization",
	Long:  "List your organizations and select one to use",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if user is logged in
		if err := utils.EnsureLoggedIn(); err != nil {
			utils.Error("%v", err)
			return
		}

		// Get userId from config
		userID := viper.GetString("userId")

		var orgID string
		var err error

		// Check if --org-id flag is provided
		if flagOrgID != "" {
			// Validate that the org exists
			orgsResp, err := api.GlobalClient.ListUserOrgs(userID)
			if err != nil {
				utils.Error("Failed to list organizations: %v", err)
				return
			}

			// Check if the provided orgId exists in the user's organizations
			orgExists := false
			for _, org := range orgsResp.Orgs {
				if org.OrgID == flagOrgID {
					orgExists = true
					break
				}
			}

			if !orgExists {
				utils.Error("Organization '%s' not found or you don't have access to it", flagOrgID)
				return
			}

			// Org exists, use it
			orgID = flagOrgID

			// Save to config
			viper.Set("orgId", orgID)
			if err := viper.WriteConfig(); err != nil {
				utils.Error("Failed to save organization to config: %v", err)
				return
			}
		} else {
			// No flag provided, use GUI selection
			orgID, err = utils.SelectOrgForm(userID)
			if err != nil {
				utils.Error("%v", err)
				return
			}
		}

		// Switch active client if running
		utils.SwitchActiveClientOrg(orgID)

		// Check if client is running and if we need to monitor a switch
		client := olm.NewClient("")
		if client.IsRunning() {
			// Get current status - if it doesn't match the new org, monitor the switch
			currentStatus, err := client.GetStatus()
			if err == nil && currentStatus != nil && currentStatus.OrgID != orgID {
				// Switch was sent, monitor the switch process
				monitorOrgSwitch(orgID)
			} else {
				// Already on the correct org or no status available
				utils.Success("Successfully selected organization: %s", orgID)
			}
		} else {
			// Client not running, no switch needed
			utils.Success("Successfully selected organization: %s", orgID)
		}
	},
}

// monitorOrgSwitch monitors the organization switch process with log preview
func monitorOrgSwitch(orgID string) {
	// Get log file path
	logFile := utils.GetDefaultLogPath()

	// Show live log preview and status during switch
	completed, err := tui.NewLogPreview(tui.LogPreviewConfig{
		LogFile: logFile,
		Header:  "Switching organization...",
		ExitCondition: func(client *olm.Client, status *olm.StatusResponse) (bool, bool) {
			// Exit when orgId matches new org AND interface is registered again
			if status != nil && status.OrgID == orgID && status.Registered {
				return true, true
			}
			return false, false
		},
		OnEarlyExit: func(client *olm.Client) {
			// User exited early - nothing to do, switch command was already sent
		},
		StatusFormatter: func(isRunning bool, status *olm.StatusResponse) string {
			if !isRunning || status == nil {
				return "Client not running"
			} else if status.OrgID == orgID && status.Registered {
				return fmt.Sprintf("Switched to %s (Registered)", orgID)
			} else if status.OrgID == orgID && !status.Registered {
				return fmt.Sprintf("Switched to %s (Registering interface)", orgID)
			} else {
				return fmt.Sprintf("Switching (current: %s)", status.OrgID)
			}
		},
	})

	// Clear the TUI lines after completion
	if completed {
		utils.Success("Successfully switched organization to: %s", orgID)
	} else if err != nil {
		utils.Warning("Failed to monitor organization switch: %v", err)
	}
}

func init() {
	orgCmd.Flags().StringVar(&flagOrgID, "org", "", "Organization ID to select")
	SelectCmd.AddCommand(orgCmd)
}
