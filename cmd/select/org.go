package selectcmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/tui"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

		// Select organization
		orgID, err := utils.SelectOrg(userID)
		if err != nil {
			utils.Error("%v", err)
			return
		}

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
	statusIconStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("46")) // Bright green (ColorSuccess)
	statusSwitchingIconStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")) // Yellow/Orange (ColorWarning)

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
				icon := statusSwitchingIconStyle.Render("○")
				return fmt.Sprintf("%s Switching...", icon)
			} else if status.OrgID == orgID && status.Registered {
				icon := statusIconStyle.Render("✓")
				return fmt.Sprintf("%s Switched to %s (Registered)", icon, orgID)
			} else if status.OrgID == orgID && !status.Registered {
				icon := statusSwitchingIconStyle.Render("○")
				return fmt.Sprintf("%s Switched to %s (Registering interface...)", icon, orgID)
			} else {
				icon := statusSwitchingIconStyle.Render("○")
				return fmt.Sprintf("%s Switching (current: %s)...", icon, status.OrgID)
			}
		},
	})

	// Clear the TUI lines after completion
	if completed {
		// Move cursor up 9 lines (header + blank + 5 logs + blank + status = 9 lines)
		fmt.Print("\033[9A\r\033[0J")
		utils.Success("Successfully switched organization to: %s", orgID)
	} else if err != nil {
		utils.Warning("Failed to monitor organization switch: %v", err)
	}
}

func init() {
	SelectCmd.AddCommand(orgCmd)
}
