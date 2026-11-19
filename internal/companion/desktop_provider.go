//go:build windows || linux || darwin

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

// desktopProvider loads the session of the Pangolin desktop app (Windows,
// Linux, or macOS). Account details live in an accounts.json of the same
// shape; secrets use the framed protocol in secrets_desktop.go. Platforms
// differ only in where the data lives and how secrets are fetched.
type desktopProvider struct {
	defaultDataDir string
	// secrets returns the client that fetches secrets for the app whose
	// data is in dataDir.
	secrets func(dataDir string) SecretsClient
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

func (p desktopProvider) DefaultDataDir() string {
	return p.defaultDataDir
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

	secretsClient := p.secrets(dataDir)
	accounts := make(map[string]config.Account, len(file.Accounts))
	for userID, desktopAccount := range file.Accounts {
		secrets, err := secretsClient.GetUserSecrets(userID)
		if err != nil {
			logger.Debug("companion: failed to load secrets for %s: %v", userID, err)
			continue
		}
		if secrets.SessionToken == "" {
			continue
		}

		account := mapDesktopAccount(desktopAccount, secrets)
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

func mapDesktopAccount(src desktopAccount, secrets UserSecrets) config.Account {
	account := config.Account{
		UserID:       src.UserID,
		Host:         src.Hostname,
		Email:        src.Email,
		OrgID:        src.OrgID,
		SessionToken: secrets.SessionToken,
	}
	if src.Username != "" {
		username := src.Username
		account.Username = &username
	}
	if src.Name != "" {
		name := src.Name
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
