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
	UserID         string          `mapstructure:"userId" json:"userId"`
	Host           string          `mapstructure:"host" json:"host"`
	Email          string          `mapstructure:"email" json:"email"`
	SessionToken   string          `mapstructure:"sessionToken" json:"sessionToken"`
	OrgID          string          `mapstructure:"orgId" json:"orgId,omitempty"`
	OlmCredentials *OlmCredentials `mapstructure:"olmCredentials" json:"olmCredentials,omitempty"`
}

type OlmCredentials struct {
	ID     string `mapstructure:"id" json:"id"`
	Secret string `mapstructure:"secret" json:"secret"`
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

	return &activeAccount, nil
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
