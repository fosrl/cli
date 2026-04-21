package api

import (
	"net/http"
)

// Default session cookie and CSRF values match the Pangolin web/API expectations.
const (
	defaultSessionCookieName = "p_session_token"
	defaultCSRFToken         = "x-csrf-protection"
)

// ClientSessionMode distinguishes how requests are authenticated.
type ClientSessionMode string

const (
	// ClientSessionModeUser is interactive login: session token sent as HTTP cookie.
	ClientSessionModeUser ClientSessionMode = "user"
	// ClientSessionModeIntegrationAPIKey is the Integration API: Bearer apiKeyId.apiKeySecret.
	ClientSessionModeIntegrationAPIKey ClientSessionMode = "integration_api_key"
)

// ClientSession holds all credentials and anti-CSRF state for outbound API calls.
// It is the single place for token, API key, cookie name, and CSRF header.
type ClientSession struct {
	Mode ClientSessionMode

	// User mode: browser-style session
	SessionToken      string
	SessionCookieName string

	// Integration mode: API key as Bearer "<id>.<secret>"
	APIKey string

	// CSRF sent as X-CSRF-Token; empty means defaultCSRFToken is used.
	CSRFToken string
}

// NewUserClientSession returns defaults for interactive / session-cookie auth.
func NewUserClientSession() ClientSession {
	return ClientSession{
		Mode:              ClientSessionModeUser,
		SessionCookieName: defaultSessionCookieName,
		CSRFToken:         defaultCSRFToken,
	}
}

// NewIntegrationAPIKeySession returns defaults for Integration API hosts (/v1/...).
func NewIntegrationAPIKeySession() ClientSession {
	return ClientSession{
		Mode:              ClientSessionModeIntegrationAPIKey,
		SessionCookieName: defaultSessionCookieName,
		CSRFToken:         defaultCSRFToken,
	}
}

func (s ClientSession) sessionCookieNameOrDefault() string {
	if s.SessionCookieName != "" {
		return s.SessionCookieName
	}
	return defaultSessionCookieName
}

func (s ClientSession) csrfValueOrDefault() string {
	if s.CSRFToken != "" {
		return s.CSRFToken
	}
	return defaultCSRFToken
}

// IsIntegrationAPIKey reports whether this session uses Integration API key auth.
func (s ClientSession) IsIntegrationAPIKey() bool {
	return s.Mode == ClientSessionModeIntegrationAPIKey
}

// HasSessionToken reports whether a user session cookie should be attached.
func (s ClientSession) HasSessionToken() bool {
	return s.SessionToken != ""
}

// HasAPIKey reports whether Bearer API key auth should be used.
func (s ClientSession) HasAPIKey() bool {
	return s.APIKey != ""
}

// ApplyToRequest sets X-CSRF-Token and authentication on the request.
// Call this for any outbound request that should match the main API client behavior.
func (s ClientSession) ApplyToRequest(req *http.Request) {
	req.Header.Set("X-CSRF-Token", s.csrfValueOrDefault())

	if s.HasSessionToken() {
		req.AddCookie(&http.Cookie{
			Name:  s.sessionCookieNameOrDefault(),
			Value: s.SessionToken,
		})
	} else if s.HasAPIKey() {
		req.Header.Set("Authorization", "Bearer "+s.APIKey)
	}
}
