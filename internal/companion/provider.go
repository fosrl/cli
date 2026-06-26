package companion

import "github.com/fosrl/cli/internal/config"

// Provider loads authentication state from a desktop application.
type Provider interface {
	Name() string
	DefaultDataDir() string
	LoadSession(dataDir string) (*Session, error)
}

// Session holds desktop app auth state mapped to CLI account types.
type Session struct {
	ActiveUserID string
	Accounts     map[string]config.Account
}

// Active returns true when the session has a logged-in active account.
func (s *Session) Active() bool {
	if s == nil || s.ActiveUserID == "" {
		return false
	}
	account, ok := s.Accounts[s.ActiveUserID]
	if !ok {
		return false
	}
	return account.SessionToken != ""
}

// PlatformAvailable reports whether companion mode is supported on this platform.
func PlatformAvailable() bool {
	return platformProvider() != nil
}

// PlatformProviderName returns the companion desktop client's OLM agent name when available.
func PlatformProviderName() string {
	provider := platformProvider()
	if provider == nil {
		return ""
	}
	return provider.Name()
}
