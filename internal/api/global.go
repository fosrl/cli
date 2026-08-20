package api

import (
	"fmt"
	"strings"
)

// InitClient initializes a new API client with stored credentials and
// a URL. The client will be created without authentication if no token
// is found. sessionCookieName overrides the default session cookie name
// when non-empty (see Config.SessionCookieName).
func InitClient(hostname string, token string, sessionCookieName string) (*Client, error) {
	// Build base URL (hostname should already include protocol from login)
	baseURL := hostname
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		// If no protocol, default to https
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/") + "/api/v1"

	if sessionCookieName == "" {
		sessionCookieName = defaultSessionCookieName
	}

	// Create API client (this should never fail, but handle it just in case)
	client, err := NewClient(ClientConfig{
		BaseURL:           baseURL,
		AgentName:         "pangolin-cli",
		Token:             token,
		SessionCookieName: sessionCookieName,
		CSRFToken:         defaultCSRFToken,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return client, nil
}

