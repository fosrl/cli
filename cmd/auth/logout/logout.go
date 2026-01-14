package logout

import (
	"errors"
	"os"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/spf13/cobra"
)

func LogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from Pangolin",
		Long:  "Logout and clear your session",
		Run: func(cmd *cobra.Command, args []string) {
			if err := logoutMain(cmd); err != nil {
				os.Exit(1)
			}
		},
	}

	return cmd
}

func logoutMain(cmd *cobra.Command) error {
	apiClient := api.FromContext(cmd.Context())

	// Check if client is running before logout
	olmClient := olm.NewClient("")
	if olmClient.IsRunning() {
		// Check that the client was started by this CLI by verifying the version
		status, err := olmClient.GetStatus()
		if err != nil {
			logger.Warning("Failed to get client status: %v", err)
			// Continue with logout even if we can't check version
		} else if status.Agent == olm.AgentName {
			// Only prompt and stop if client was started by this CLI
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
				logger.Error("Error: %v", err)
				return err
			}

			if !confirm {
				err := errors.New("logout cancelled")
				logger.Info("%v", err)
				return err
			}

			// Kill the client without showing TUI
			_, err := olmClient.Exit()
			if err != nil {
				logger.Warning("Failed to send exit signal to client: %v", err)
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
					logger.Warning("Client did not stop within timeout")
				}
			}
		}
		// If version doesn't match, skip client shutdown and continue with logout
	}

	// Check if there's an active session in the key store
	accountStore, err := config.LoadAccountStore()
	if err != nil {
		logger.Error("Failed to load account store: %s", err)
		return err
	}

	if accountStore.ActiveUserID == "" {
		logger.Success("Already logged out!")
		return nil
	}

	// Try to logout from server (client is always initialized)
	if err := apiClient.Logout(); err != nil {
		// Ignore logout errors - we'll still clear local data
		logger.Debug("Failed to logout from server: %v", err)
	}

	deletedAccount := accountStore.Accounts[accountStore.ActiveUserID]
	delete(accountStore.Accounts, accountStore.ActiveUserID)

	// If there are still other accounts, then we need to set the active key again.
	// Automatically set next active user ID to the first account found.
	if nextUserID, ok := anyKey(accountStore.Accounts); ok {
		accountStore.ActiveUserID = nextUserID
	} else {
		accountStore.ActiveUserID = ""
	}

	if err := accountStore.Save(); err != nil {
		logger.Error("Failed to save account store: %v", err)
		return err
	}

	// Print logout message with account name
	logger.Success("Logged out of Pangolin account %s", deletedAccount.Email)

	return nil
}

func anyKey[K comparable, V any](m map[K]V) (K, bool) {
	var zero K
	for k := range m {
		return k, true
	}
	return zero, false
}
