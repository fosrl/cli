//go:build windows

package companion

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
)

const windowsAppName = "Pangolin"
const accountsFileName = "accounts.json"

type windowsProvider struct {
	secrets SecretsClient
}

type windowsAccountsFile struct {
	ActiveUserID string                    `json:"activeUserId"`
	Accounts     map[string]windowsAccount `json:"accounts"`
}

type windowsAccount struct {
	UserID   string `json:"userId"`
	Email    string `json:"email"`
	OrgID    string `json:"orgId"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
}

func (windowsProvider) Name() string {
	return olm.CompanionAgentName
}

func (windowsProvider) DefaultDataDir() string {
	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		appData = os.Getenv("APPDATA")
	}
	return filepath.Join(appData, windowsAppName)
}

func (p windowsProvider) LoadSession(dataDir string) (*Session, error) {
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

	var file windowsAccountsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if file.Accounts == nil {
		return nil, nil
	}

	accounts := make(map[string]config.Account, len(file.Accounts))
	for userID, winAccount := range file.Accounts {
		secrets, err := p.secrets.GetUserSecrets(userID)
		if err != nil {
			logger.Debug("companion: failed to load secrets for %s: %v", userID, err)
			continue
		}
		if secrets.SessionToken == "" {
			continue
		}

		account := mapWindowsAccount(winAccount, secrets)
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

func mapWindowsAccount(win windowsAccount, secrets UserSecrets) config.Account {
	account := config.Account{
		UserID:       win.UserID,
		Host:         win.Hostname,
		Email:        win.Email,
		OrgID:        win.OrgID,
		SessionToken: secrets.SessionToken,
	}
	if win.Username != "" {
		username := win.Username
		account.Username = &username
	}
	if win.Name != "" {
		name := win.Name
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

func platformProvider() Provider {
	return windowsProvider{secrets: newPipeSecretsClient()}
}
