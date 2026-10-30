package utils

import (
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
)

// RefreshClientSettings fetches the settings the org manages for the active
// account and records them in the account store. It runs every time the client
// connects and whenever settings are looked at. If the server can't be reached
// the last fetched settings keep applying.
func RefreshClientSettings(fetch func(orgID string) (*config.ClientSettings, error), accountStore *config.AccountStore, account *config.Account, orgID string) *config.ClientSettings {
	settings, err := fetch(orgID)
	if err != nil {
		logger.Warning("Could not fetch client settings from the server (%v); using the last known settings", err)
		return account.ServerSettings(orgID)
	}

	account.SetServerSettings(orgID, settings)
	if accountStore.IsReadOnly() {
		return settings
	}
	err = accountStore.UpdateActiveAccount(account)
	if err == nil {
		err = accountStore.Save()
	}
	if err != nil {
		logger.Warning("Could not save client settings: %v", err)
	}
	return settings
}
