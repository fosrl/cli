package org

import (
	"fmt"
	"os"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/tui"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

type OrgCmdOpts struct {
	OrgID string
}

func OrgCmd() *cobra.Command {
	opts := OrgCmdOpts{}

	cmd := &cobra.Command{
		Use:   "org",
		Short: "Select an organization",
		Long:  "List your organizations and select one to use",
		Run: func(cmd *cobra.Command, args []string) {
			if err := orgMain(cmd, &opts); err != nil {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&opts.OrgID, "org", "", "Organization `ID` to select")

	_ = cmd.RegisterFlagCompletionFunc("org", completeOrgID)

	return cmd
}

func orgMain(cmd *cobra.Command, opts *OrgCmdOpts) error {
	apiClient := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())
	cfg := config.ConfigFromContext(cmd.Context())

	activeAccount, err := accountStore.ActiveAccount()
	if err != nil {
		logger.Error("%v", err)
		return err

	}
	userID := activeAccount.UserID

	var selectedOrgID string

	// Check if --org-id flag is provided
	if opts.OrgID != "" {
		// Validate that the org exists
		orgsResp, err := apiClient.ListUserOrgs(userID)
		if err != nil {
			logger.Error("Failed to list organizations: %v", err)
			return err
		}

		// Check if the provided orgId exists in the user's organizations
		orgExists := false
		for _, org := range orgsResp.Orgs {
			if org.OrgID == opts.OrgID {
				orgExists = true
				break
			}
		}

		if !orgExists {
			err := fmt.Errorf("organization '%s' not found or you don't have access to it", opts.OrgID)
			logger.Error("Error: %v", err)
			return err
		}

		// Org exists, use it
		selectedOrgID = opts.OrgID
	} else {
		// No flag provided, use GUI selection
		selectedOrgID, err = utils.SelectOrgForm(apiClient, userID)
		if err != nil {
			logger.Error("%v", err)
			return err
		}
	}

	activeAccount.OrgID = selectedOrgID
	if err := accountStore.Save(); err != nil {
		logger.Error("Failed to save account to store: %v", err)
		return err
	}

	// Switch active client if running
	utils.SwitchActiveClientOrg(selectedOrgID)

	// Check if olmClient is running and if we need to monitor a switch
	olmClient := olm.NewClient("")
	if olmClient.IsRunning() {
		// Get current status - if it doesn't match the new org, monitor the switch
		currentStatus, err := olmClient.GetStatus()
		if err == nil && currentStatus != nil && currentStatus.OrgID != selectedOrgID {
			// Switch was sent, monitor the switch process
			monitorOrgSwitch(cfg.LogFile, selectedOrgID)
		} else {
			// Already on the correct org or no status available
			logger.Success("Successfully selected organization: %s", selectedOrgID)
		}
	} else {
		// Client not running, no switch needed
		logger.Success("Successfully selected organization: %s", selectedOrgID)
	}

	return nil
}

// monitorOrgSwitch monitors the organization switch process with log preview
func monitorOrgSwitch(logFile string, orgID string) {
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
		logger.Success("Successfully switched organization to: %s", orgID)
	} else if err != nil {
		logger.Warning("Failed to monitor organization switch: %v", err)
	}
}
