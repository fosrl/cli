package companion

import (
	"fmt"

	"github.com/fosrl/cli/internal/config"
)

// State describes the resolved companion mode for the current process.
type State struct {
	Enabled            bool
	Active             bool
	ProviderName       string
	Session            *Session
	NotReadySuggestion string
}

// ResolveDataDir returns the desktop app data directory from config or platform default.
func ResolveDataDir(cfg *config.Config, provider Provider) string {
	if override := cfg.CompanionAppDataDirForPlatform(); override != "" {
		return override
	}
	if provider != nil {
		return provider.DefaultDataDir()
	}
	return ""
}

// Resolve loads companion state and the account store to use for the CLI.
func Resolve(cfg *config.Config) (*State, *config.AccountStore, error) {
	provider := platformProvider()
	state := &State{}

	if cfg.DisableCompanionMode || provider == nil {
		store, err := config.LoadAccountStore(cfg)
		return state, store, err
	}

	state.Enabled = true
	state.ProviderName = provider.Name()

	dataDir := ResolveDataDir(cfg, provider)
	session, err := provider.LoadSession(dataDir)
	if err != nil {
		return state, nil, err
	}

	if session != nil && session.Active() {
		state.Active = true
		state.Session = session
		store := config.NewReadOnlyAccountStore(session.ActiveUserID, session.Accounts)
		return state, store, nil
	}

	state.NotReadySuggestion = notReadySuggestion(provider, dataDir, session)

	// Companion enabled but desktop app is not logged in.
	store := config.NewReadOnlyAccountStore("", map[string]config.Account{})
	return state, store, nil
}

func notReadySuggestion(provider Provider, dataDir string, session *Session) string {
	if session != nil && session.Active() {
		return ""
	}

	activeUserID, accountCount := desktopAccountsHint(dataDir)
	if activeUserID != "" && session == nil {
		if RequiredDesktopAppVersion() != "" {
			return fmt.Sprintf(
				"Start %s (version %s or later) and log in, or run 'pangolin companion disable'.",
				provider.Name(),
				RequiredDesktopAppVersion(),
			)
		}
		return fmt.Sprintf(
			"Start %s and log in, or run 'pangolin companion disable'.",
			provider.Name(),
		)
	}

	if accountCount > 0 || activeUserID != "" {
		return fmt.Sprintf("Open %s and log in.", provider.Name())
	}

	return fmt.Sprintf("Open %s and log in.", provider.Name())
}

// MutatingAuthError is returned when auth commands are blocked in companion mode.
func MutatingAuthError(providerName string) error {
	return fmt.Errorf(
		"Authentication is managed by %s.\n"+
			"Login, logout, and account or organization changes must be done in %s.\n"+
			"To use standalone CLI auth, run 'pangolin companion disable'.",
		providerName,
		providerName,
	)
}
