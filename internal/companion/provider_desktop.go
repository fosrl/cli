//go:build windows || linux

package companion

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
)

const accountsFileName = "accounts.json"

// desktopProvider reads the Pangolin desktop client's session, on Windows and
// Linux alike: account details from its accounts.json, and secrets from its
// manager service, which only the manager can read.
type desktopProvider struct {
	secrets SecretsClient
}

type desktopAccountsFile struct {
	ActiveUserID string                    `json:"activeUserId"`
	Accounts     map[string]desktopAccount `json:"accounts"`
}

type desktopAccount struct {
	UserID   string `json:"userId"`
	Email    string `json:"email"`
	OrgID    string `json:"orgId"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
}

func (desktopProvider) Name() string {
	return olm.CompanionAgentName
}

func (desktopProvider) DefaultDataDir() string {
	return defaultDesktopDataDir()
}

func (p desktopProvider) LoadSession(dataDir string) (*Session, error) {
	if dataDir == "" {
		return nil, nil
	}

	accountsPath := filepath.Join(dataDir, accountsFileName)
	data, err := os.ReadFile(accountsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var file desktopAccountsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if file.Accounts == nil {
		return nil, nil
	}

	accounts := make(map[string]config.Account, len(file.Accounts))
	for userID, appAccount := range file.Accounts {
		secrets, err := p.secrets.GetUserSecrets(userID)
		if err != nil {
			logger.Debug("companion: failed to load secrets for %s: %v", userID, err)
			continue
		}
		if secrets.SessionToken == "" {
			continue
		}

		account := mapDesktopAccount(appAccount, secrets)
		accounts[userID] = account
	}

	if len(accounts) == 0 {
		return nil, nil
	}

	activeUserID := file.ActiveUserID
	if activeUserID != "" {
		if _, ok := accounts[activeUserID]; !ok {
			activeUserID = ""
		}
	}
	if activeUserID == "" {
		for userID := range accounts {
			activeUserID = userID
			break
		}
	}

	session := &Session{
		ActiveUserID: activeUserID,
		Accounts:     accounts,
	}
	if !session.Active() {
		return nil, nil
	}
	return session, nil
}

func mapDesktopAccount(app desktopAccount, secrets UserSecrets) config.Account {
	account := config.Account{
		UserID:       app.UserID,
		Host:         app.Hostname,
		Email:        app.Email,
		OrgID:        app.OrgID,
		SessionToken: secrets.SessionToken,
	}
	if app.Username != "" {
		username := app.Username
		account.Username = &username
	}
	if app.Name != "" {
		name := app.Name
		account.Name = &name
	}
	if secrets.OlmID != "" || secrets.OlmSecret != "" {
		account.OlmCredentials = &config.OlmCredentials{
			ID:     secrets.OlmID,
			Secret: secrets.OlmSecret,
		}
	}
	return account
}
