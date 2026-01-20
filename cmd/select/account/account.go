package account

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

type AccountCmdOpts struct {
	Account string
	Host    string
}

func AccountCmd() *cobra.Command {
	opts := AccountCmdOpts{}

	cmd := &cobra.Command{
		Use:   "account",
		Short: "Select an account",
		Long:  "List your logged-in accounts and select active one",
		Run: func(cmd *cobra.Command, args []string) {
			if err := accountMain(cmd, &opts); err != nil {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&opts.Account, "account", "a", "", "Account to select")
	cmd.Flags().StringVar(&opts.Host, "host", "", "Pangolin host where account is located")

	_ = cmd.RegisterFlagCompletionFunc("account", completeAccountFlag)
	_ = cmd.RegisterFlagCompletionFunc("host", completeHostFlag)

	return cmd
}

func accountMain(cmd *cobra.Command, opts *AccountCmdOpts) error {
	accountStore := config.AccountStoreFromContext(cmd.Context())

	if len(accountStore.Accounts) == 0 {
		err := errors.New("not logged in")
		logger.Error("Error: %v", err)
		return err
	}

	var selectedAccount *config.Account

	// If flag is provided, find an account that matches the
	// terms verbatim.
	if opts.Account != "" {
		for _, account := range accountStore.Accounts {
			if opts.Host != "" && opts.Host != account.Host {
				continue
			}

			if opts.Account == account.Email {
				selectedAccount = &account
				break
			}
		}

		if selectedAccount == nil {
			err := errors.New("no accounts found that match the search terms")
			logger.Error("Error: %v", err)
			return err
		}
	} else {
		// No flag provided, use GUI selection if necessary
		selected, err := selectAccountForm(accountStore.Accounts, opts.Host)
		if err != nil {
			logger.Error("Error: failed to select account: %v", err)
			return err
		}

		selectedAccount = selected
	}

	// Optimistic account switching: switch locally first
	apiClient := api.FromContext(cmd.Context())
	
	// 1. Switch account locally first
	accountStore.ActiveUserID = selectedAccount.UserID
	
	// Update API client base URL and token from account
	apiBaseURL := selectedAccount.Host
	if apiBaseURL != "" {
		// Ensure it has /api/v1 suffix
		if !strings.HasSuffix(apiBaseURL, "/api/v1") {
			if !strings.HasSuffix(apiBaseURL, "/") {
				apiBaseURL = apiBaseURL + "/api/v1"
			} else {
				apiBaseURL = apiBaseURL + "api/v1"
			}
		}
		apiClient.SetBaseURL(apiBaseURL)
	}
	apiClient.SetToken(selectedAccount.SessionToken)
	
	if err := accountStore.Save(); err != nil {
		logger.Error("Error: failed to save account to store: %v", err)
		return err
	}

	// Check if olmClient is running and if we need to shut it down
	olmClient := olm.NewClient("")
	if olmClient.IsRunning() {
		logger.Info("Shutting down running client")
		_, err := olmClient.Exit()
		if err != nil {
			logger.Warning("Failed to shut down OLM client: %s; you may need to do so manually.", err)
		}
	}

	// 2. Then validate with server
	// Check health before fetching user data
	healthOk, healthErr := apiClient.CheckHealth()
	if healthErr != nil || !healthOk {
		logger.Warning("The server appears to be down. Account switched, but unable to verify with server.")
		// Still show success message using stored account data
		selectedAccountStr := utils.AccountDisplayNameWithHost(selectedAccount)
		logger.Success("Successfully selected account: %s", selectedAccountStr)
		return nil
	}

	// Health check passed, fetch user data and update account info
	user, err := apiClient.GetUser()
	if err != nil {
		logger.Warning("Failed to fetch user data: %v. Account switched, but user info not updated.", err)
		// Still show success message using stored account data
		selectedAccountStr := utils.AccountDisplayNameWithHost(selectedAccount)
		logger.Success("Successfully selected account: %s", selectedAccountStr)
		return nil
	}

	// Update account with username and name from user data
	if user.Username != nil || user.Name != nil {
		username := ""
		name := ""
		if user.Username != nil {
			username = *user.Username
		}
		if user.Name != nil {
			name = *user.Name
		}
		if err := accountStore.UpdateAccountUserInfo(selectedAccount.UserID, username, name); err != nil {
			logger.Debug("Failed to update account user info: %v", err)
		}
		// Reload selected account to get updated info
		updatedAccount, exists := accountStore.Accounts[selectedAccount.UserID]
		if exists {
			selectedAccount = &updatedAccount
		}
	}

	// Fetch server info
	apiServerInfo, err := apiClient.GetServerInfo()
	if err != nil {
		logger.Debug("Failed to fetch server info: %v", err)
	} else if apiServerInfo != nil {
		// Convert api.ServerInfo to config.ServerInfo
		serverInfo := &config.ServerInfo{
			Version:                  apiServerInfo.Version,
			SupporterStatusValid:     apiServerInfo.SupporterStatusValid,
			Build:                    apiServerInfo.Build,
			EnterpriseLicenseValid:   apiServerInfo.EnterpriseLicenseValid,
			EnterpriseLicenseType:    apiServerInfo.EnterpriseLicenseType,
		}
		// Update account with server info
		account := accountStore.Accounts[selectedAccount.UserID]
		account.ServerInfo = serverInfo
		accountStore.Accounts[selectedAccount.UserID] = account
		if err := accountStore.Save(); err != nil {
			logger.Debug("Failed to save server info: %v", err)
		}
		// Reload selected account to get updated info
		updatedAccount, exists := accountStore.Accounts[selectedAccount.UserID]
		if exists {
			selectedAccount = &updatedAccount
		}
	}

	selectedAccountStr := utils.AccountDisplayNameWithHost(selectedAccount)
	logger.Success("Successfully selected account: %s", selectedAccountStr)

	return nil
}

// selectAccountForm lists organizations for a user and prompts them to select one.
// It returns the selected org ID and any error.
// If the user has only one organization, it's automatically selected.
func selectAccountForm(accounts map[string]config.Account, hostFilter string) (*config.Account, error) {
	var filteredAccounts []*config.Account
	for _, account := range accounts {
		if hostFilter == "" || hostFilter == account.Host {
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
		label := utils.AccountDisplayNameWithHost(account)
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
