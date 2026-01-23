package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type AccountStore struct {
	// All operations must happen to the configuration file,
	// so they must operate on separate Viper instances.
	v *viper.Viper

	ActiveUserID string             `mapstructure:"activeUserId" json:"activeUserId"`
	Accounts     map[string]Account `mapstructure:"accounts" json:"accounts"`
}

type Account struct {
	UserID         string           `mapstructure:"userId" json:"userId"`
	Host           string           `mapstructure:"host" json:"host"`
	Email          string           `mapstructure:"email" json:"email"`
	Username       *string          `mapstructure:"username" json:"username,omitempty"`
	Name           *string          `mapstructure:"name" json:"name,omitempty"`
	SessionToken   string           `mapstructure:"sessionToken" json:"sessionToken"`
	OrgID          string           `mapstructure:"orgId" json:"orgId,omitempty"`
	OlmCredentials *OlmCredentials `mapstructure:"olmCredentials" json:"olmCredentials,omitempty"`
	ServerInfo     *ServerInfo     `mapstructure:"serverInfo" json:"serverInfo,omitempty"`
}

type OlmCredentials struct {
	ID     string `mapstructure:"id" json:"id"`
	Secret string `mapstructure:"secret" json:"secret"`
}

// ServerInfo represents server information including version, build type, and license status
// This mirrors api.ServerInfo to avoid import cycles
type ServerInfo struct {
	Version                  string  `mapstructure:"version" json:"version"`
	SupporterStatusValid     bool    `mapstructure:"supporterStatusValid" json:"supporterStatusValid"`
	Build                    string  `mapstructure:"build" json:"build"` // "oss" | "enterprise" | "saas"
	EnterpriseLicenseValid   bool    `mapstructure:"enterpriseLicenseValid" json:"enterpriseLicenseValid"`
	EnterpriseLicenseType    *string `mapstructure:"enterpriseLicenseType" json:"enterpriseLicenseType,omitempty"`
}

func newAccountViper() (*viper.Viper, error) {
	v := viper.New()

	dir, err := GetPangolinConfigDir()
	if err != nil {
		return nil, err
	}

	accountsFile := filepath.Join(dir, "accounts.json")
	v.SetConfigFile(accountsFile)
	v.SetConfigType("json")

	return v, nil
}

func LoadAccountStore() (*AccountStore, error) {
	v, err := newAccountViper()
	if err != nil {
		return nil, err
	}

	store := AccountStore{
		v:            v,
		ActiveUserID: "",
		Accounts:     map[string]Account{},
	}

	if err := v.ReadInConfig(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &store, nil
		}
		return nil, err
	}

	if err := v.Unmarshal(&store); err != nil {
		return nil, err
	}

	return &store, nil
}

func (s *AccountStore) ActiveAccount() (*Account, error) {
	if s.ActiveUserID == "" {
		return nil, errors.New("not logged in")
	}

	activeAccount, exists := s.Accounts[s.ActiveUserID]
	if !exists {
		return nil, errors.New("active account missing")
	}

	if activeAccount.SessionToken == "" {
		return nil, errors.New("active account missing session token")
	}

	return &activeAccount, nil
}

// Set account with the user ID as "inactive"; keeps the Olm
// credentials for the account, but clear other account state
// like the session token and selected org ID.
//
// This effectively logs out the account.
func (s *AccountStore) Deactivate(userID string) error {
	account, exists := s.Accounts[userID]
	if !exists {
		return errors.New("account does not exist")
	}

	account.SessionToken = ""
	account.OrgID = ""

	s.Accounts[userID] = account

	if s.ActiveUserID == userID {
		s.ActiveUserID = ""
	}

	return s.Save()
}

// Return a list of accounts that are available to use.
// These accounts are guaranteed to have a valid
// session token.
func (s *AccountStore) AvailableAccounts() []Account {
	available := []Account{}

	for _, account := range s.Accounts {
		if account.SessionToken != "" {
			available = append(available, account)
		}
	}

	return available
}

// UpdateActiveAccount updates the active account in the store.
// This must be called after modifying an account obtained from ActiveAccount()
// because Go maps return copies of values, not references.
func (s *AccountStore) UpdateActiveAccount(account *Account) error {
	if s.ActiveUserID == "" {
		return errors.New("not logged in")
	}

	if account.UserID != s.ActiveUserID {
		return errors.New("account user ID does not match active user ID")
	}

	s.Accounts[s.ActiveUserID] = *account
	return nil
}

func (s *AccountStore) Save() error {
	// HACK: If there's a better way to write the config all at once
	// without having to specify each toplevel struct key, that
	// would be preferable.
	// However, this is fine for now.
	s.v.Set("activeUserId", s.ActiveUserID)
	s.v.Set("accounts", s.Accounts)

	return s.v.WriteConfig()
}

// UpdateAccountUserInfo updates the username and name for a specific account
func (s *AccountStore) UpdateAccountUserInfo(userID, username, name string) error {
	account, exists := s.Accounts[userID]
	if !exists {
		return errors.New("account not found")
	}

	if username != "" {
		account.Username = &username
	}
	if name != "" {
		account.Name = &name
	}

	s.Accounts[userID] = account
	return s.Save()
}
