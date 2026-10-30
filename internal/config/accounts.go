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

	readOnly bool

	ActiveUserID string             `mapstructure:"activeUserId" json:"activeUserId"`
	Accounts     map[string]Account `mapstructure:"accounts" json:"accounts"`
}

type Account struct {
	UserID         string          `mapstructure:"userId" json:"userId"`
	Host           string          `mapstructure:"host" json:"host"`
	Email          string          `mapstructure:"email" json:"email"`
	Username       *string         `mapstructure:"username" json:"username,omitempty"`
	Name           *string         `mapstructure:"name" json:"name,omitempty"`
	SessionToken   string          `mapstructure:"sessionToken" json:"sessionToken"`
	OrgID          string          `mapstructure:"orgId" json:"orgId,omitempty"`
	OlmCredentials *OlmCredentials `mapstructure:"olmCredentials" json:"olmCredentials,omitempty"`
	ServerInfo     *ServerInfo     `mapstructure:"serverInfo" json:"serverInfo,omitempty"`

	OrgSettings map[string]OrgSettings `mapstructure:"orgSettings" json:"orgSettings,omitempty"`
}

type OrgSettings struct {
	// The exit node (a gateway-mode site resource) selected with `pangolin
	// select exit-node`, re-applied by `pangolin up` for this org. Only the
	// resource ID is stored (not the niceId, which can be renamed); its sites
	// are looked up from the server on every start so they can't go stale.
	ExitNodeResourceID int `mapstructure:"exitNodeResourceId" json:"exitNodeResourceId,omitempty"`

	// The settings the org's server manages for this user, as last fetched
	// from it. Refreshed by the server on every connect; only kept here so they
	// still apply if the server can't be reached.
	ServerSettings *ClientSettings `mapstructure:"serverSettings" json:"serverSettings,omitempty"`
}

func (s OrgSettings) IsEmpty() bool {
	return s.ExitNodeResourceID == 0 && s.ServerSettings == nil
}

// ClientSettings is the server's GET /org/:orgId/my-client-settings response.
// A nil/empty field isn't managed by the server.
type ClientSettings struct {
	OverrideDNS             *bool    `mapstructure:"overrideDns" json:"overrideDns,omitempty"`
	MTU                     *int     `mapstructure:"mtu" json:"mtu,omitempty"`
	TunnelDNS               *bool    `mapstructure:"tunnelDns" json:"tunnelDns,omitempty"`
	PrimaryDNS              string   `mapstructure:"primaryDns" json:"primaryDns,omitempty"`
	SecondaryDNS            string   `mapstructure:"secondaryDns" json:"secondaryDns,omitempty"`
	ExitNodeTakesPrecedence *bool    `mapstructure:"exitNodeTakesPrecedence" json:"exitNodeTakesPrecedence,omitempty"`
	ExitNodeSiteResourceID  int      `mapstructure:"exitNodeSiteResourceId" json:"exitNodeSiteResourceId,omitempty"`
	ExitNodeLocked          bool     `mapstructure:"exitNodeLocked" json:"exitNodeLocked,omitempty"`
	MatchDomains            []string `mapstructure:"matchDomains" json:"matchDomains,omitempty"`
	PreferLocalRoutes       *bool    `mapstructure:"preferLocalRoutes" json:"preferLocalRoutes,omitempty"`
}

// LockedExitNode returns the exit node the server forces, or 0 if it forces
// none.
func (s *ClientSettings) LockedExitNode() int {
	if s == nil || !s.ExitNodeLocked {
		return 0
	}
	return s.ExitNodeSiteResourceID
}

// UpstreamDNS returns the managed primary and secondary DNS servers, if any.
func (s *ClientSettings) UpstreamDNS() []string {
	if s == nil {
		return nil
	}
	var servers []string
	for _, server := range []string{s.PrimaryDNS, s.SecondaryDNS} {
		if server != "" {
			servers = append(servers, server)
		}
	}
	return servers
}

// ManagesKey reports whether the server manages the given config key (as named
// by ConfigOptions).
func (s *ClientSettings) ManagesKey(key string) bool {
	if s == nil {
		return false
	}
	switch key {
	case "up.tunnel_dns":
		return s.TunnelDNS != nil
	case "up.override_dns":
		return s.OverrideDNS != nil
	case "up.upstream_dns":
		return len(s.UpstreamDNS()) > 0
	case "up.match_domains_dns":
		return len(s.MatchDomains) > 0
	case "up.prefer_local_routes":
		return s.PreferLocalRoutes != nil
	case "up.exit_node_takes_precedence":
		return s.ExitNodeTakesPrecedence != nil
	case "up.mtu":
		return s.MTU != nil
	case "up.exit_node_resource_id":
		return s.LockedExitNode() != 0
	default:
		return false
	}
}

// EffectiveExitNode picks the exit node to use for an org: one the server locks
// always wins; next the user's own selection (savedResourceID), then the org's
// entry in the config file, and last the server's default.
func EffectiveExitNode(server *ClientSettings, savedResourceID int, org OrgConfig) int {
	if locked := server.LockedExitNode(); locked != 0 {
		return locked
	}
	if savedResourceID != 0 {
		return savedResourceID
	}
	if org.ExitNodeResourceID != nil {
		return *org.ExitNodeResourceID
	}
	if server != nil {
		return server.ExitNodeSiteResourceID
	}
	return 0
}

// GetOrgSettings returns the account's settings for orgID, or empty defaults
// if none are stored.
func (a *Account) GetOrgSettings(orgID string) OrgSettings {
	return a.OrgSettings[orgID]
}

// UpdateOrgSettings updates the account's settings for orgID. Empty settings
// remove that org's entry.
func (a *Account) UpdateOrgSettings(orgID string, update func(*OrgSettings)) {
	settings := make(map[string]OrgSettings, len(a.OrgSettings)+1)
	for id, s := range a.OrgSettings {
		settings[id] = s
	}
	org := settings[orgID]
	update(&org)
	if org.IsEmpty() {
		delete(settings, orgID)
	} else {
		settings[orgID] = org
	}
	if len(settings) == 0 {
		settings = nil
	}
	a.OrgSettings = settings
}

// ServerSettings returns the settings last fetched from the server for orgID,
// or nil if it manages none.
func (a *Account) ServerSettings(orgID string) *ClientSettings {
	return a.GetOrgSettings(orgID).ServerSettings
}

// SetServerSettings records the settings fetched from the server for orgID.
func (a *Account) SetServerSettings(orgID string, settings *ClientSettings) {
	a.UpdateOrgSettings(orgID, func(s *OrgSettings) {
		s.ServerSettings = settings
	})
}

// ExitNode returns the resource ID of the exit node selected for orgID, or 0
// for none.
func (a *Account) ExitNode(orgID string) int {
	return a.GetOrgSettings(orgID).ExitNodeResourceID
}

// SetExitNode records the exit node (a gateway resource) selected for orgID.
func (a *Account) SetExitNode(orgID string, resourceID int) {
	a.UpdateOrgSettings(orgID, func(s *OrgSettings) {
		s.ExitNodeResourceID = resourceID
	})
}

// ClearExitNode forgets the exit node selected for orgID.
func (a *Account) ClearExitNode(orgID string) {
	a.SetExitNode(orgID, 0)
}

type OlmCredentials struct {
	ID     string `mapstructure:"id" json:"id"`
	Secret string `mapstructure:"secret" json:"secret"`
}

// ServerInfo represents server information including version, build type, and license status
// This mirrors api.ServerInfo to avoid import cycles
type ServerInfo struct {
	Version                string  `mapstructure:"version" json:"version"`
	SupporterStatusValid   bool    `mapstructure:"supporterStatusValid" json:"supporterStatusValid"`
	Build                  string  `mapstructure:"build" json:"build"` // "oss" | "enterprise" | "saas"
	EnterpriseLicenseValid bool    `mapstructure:"enterpriseLicenseValid" json:"enterpriseLicenseValid"`
	EnterpriseLicenseType  *string `mapstructure:"enterpriseLicenseType" json:"enterpriseLicenseType,omitempty"`
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

func LoadAccountStore(cfg *Config) (*AccountStore, error) {
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

// NewReadOnlyAccountStore builds an in-memory account store from desktop app session data.
func NewReadOnlyAccountStore(activeUserID string, accounts map[string]Account) *AccountStore {
	if accounts == nil {
		accounts = map[string]Account{}
	}
	return &AccountStore{
		readOnly:     true,
		ActiveUserID: activeUserID,
		Accounts:     accounts,
	}
}

// IsReadOnly reports whether the store is managed by companion mode and cannot be persisted.
func (s *AccountStore) IsReadOnly() bool {
	return s.readOnly
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
	if s.readOnly {
		return errors.New("account store is read-only")
	}
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
	if s.readOnly {
		return errors.New("account store is read-only")
	}
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
	if s.readOnly {
		return errors.New("account store is read-only")
	}
	if s.v == nil {
		return errors.New("account store has no backing config file")
	}
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
	if s.readOnly {
		return errors.New("account store is read-only")
	}
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
