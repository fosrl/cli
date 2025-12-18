package account

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/spf13/cobra"
)

var (
	accountToSelect string
	hostToSelect    string
)

var AccountCmd = &cobra.Command{
	Use:   "account",
	Short: "Select an account",
	Long:  "List your logged-in accounts and select active one",
	Run: func(cmd *cobra.Command, args []string) {
		accountStore := config.AccountStoreFromContext(cmd.Context())

		if len(accountStore.Accounts) == 0 {
			logger.Warning("Not logged in.")
			return
		}

		var selectedAccount *config.Account

		// If flag is provided, find an account that matches the
		// terms verbatim.
		if accountToSelect != "" {
			for _, account := range accountStore.Accounts {
				if hostToSelect != "" && hostToSelect != account.Host {
					continue
				}

				if accountToSelect == account.Email {
					selectedAccount = &account
					break
				}
			}

			if selectedAccount == nil {
				logger.Error("No accounts found that match the search terms")
				return
			}
		} else {
			// No flag provided, use GUI selection if necessary
			selected, err := selectAccountForm(accountStore.Accounts)
			if err != nil {
				logger.Error("Failed to select account: %v", err)
				return
			}

			selectedAccount = selected
		}

		accountStore.ActiveUserID = selectedAccount.UserID
		if err := accountStore.Save(); err != nil {
			logger.Error("Failed to save account to store: %v", err)
			return
		}

		// Check if olmClient is running and if we need to shut it down
		olmClient := olm.NewClient("")
		if olmClient.IsRunning() {
			logger.Info("Shutting down running OLM client")
			_, err := olmClient.Exit()
			if err != nil {
				logger.Warning("Failed to shut down OLM client: %s; you may need to do so manually.", err)
			}
		}

		selectedAccountStr := fmt.Sprintf("%s @ %s", selectedAccount.Email, selectedAccount.Host)
		logger.Success("Successfully selected account: %s", selectedAccountStr)
	},
}

// selectAccountForm lists organizations for a user and prompts them to select one.
// It returns the selected org ID and any error.
// If the user has only one organization, it's automatically selected.
func selectAccountForm(accounts map[string]config.Account) (*config.Account, error) {
	var filteredAccounts []*config.Account
	for _, account := range accounts {
		if hostToSelect == "" || hostToSelect == account.Host {
			filteredAccounts = append(filteredAccounts, &account)
		}
	}

	if len(filteredAccounts) == 0 {
		return nil, fmt.Errorf("no accounts found that match the query")
	}

	if len(filteredAccounts) == 1 {
		// Auto-select the first account
		for _, account := range filteredAccounts {
			return account, nil
		}
	}

	type accountOption struct {
		Account *config.Account
		Label   string
	}

	var orgOptions []huh.Option[accountOption]
	for _, account := range filteredAccounts {
		label := fmt.Sprintf("%s @ %s", account.Email, account.Host)
		orgOptions = append(orgOptions, huh.NewOption(label, accountOption{
			Account: account,
			Label:   label,
		}))
	}

	var selectedAccountOption accountOption
	orgSelectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[accountOption]().
				Title("Select an account").
				Options(orgOptions...).
				Value(&selectedAccountOption),
		),
	)

	if err := orgSelectForm.Run(); err != nil {
		return nil, fmt.Errorf("error running account selection form: %w", err)
	}

	return selectedAccountOption.Account, nil
}

func init() {
	AccountCmd.Flags().StringVarP(&accountToSelect, "account", "a", "", "Account to select")
	AccountCmd.Flags().StringVar(&hostToSelect, "host", "", "Pangolin host where account is located")
}
