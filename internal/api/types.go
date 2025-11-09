package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Client represents the API client configuration
type Client struct {
	BaseURL           string
	AgentName         string
	APIKey            string
	Token             string
	SessionCookieName string
	CSRFToken         string
	HTTPClient        *HTTPClient
}

// HTTPClient wraps the standard http.Client with additional configuration
type HTTPClient struct {
	Timeout time.Duration
}

// RequestOptions contains optional parameters for API requests
type RequestOptions struct {
	Headers map[string]string
	Query   map[string]string
}

// FlexibleBool can unmarshal from both boolean and string JSON values
type FlexibleBool bool

func (b *FlexibleBool) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case bool:
		*b = FlexibleBool(value)
	case string:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			// If string is non-empty, treat as true (error condition)
			*b = FlexibleBool(value != "" && value != "false")
		} else {
			*b = FlexibleBool(boolValue)
		}
	default:
		*b = false
	}
	return nil
}

func (b FlexibleBool) Bool() bool {
	return bool(b)
}

// APIResponse represents the standard API response format
type APIResponse struct {
	Data    json.RawMessage `json:"data"`
	Success bool            `json:"success"`
	Error   FlexibleBool    `json:"error"`
	Message string          `json:"message"`
	Status  int             `json:"status"`
	Stack   string          `json:"stack,omitempty"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
	Stack   string `json:"stack,omitempty"`
}

// Error implements the error interface
// Returns just the message if present, otherwise just the status code
func (e *ErrorResponse) Error() string {
	if e.Message != "" {
		return e.Message
	}
	// If no message, return just the status code
	return fmt.Sprintf("%d", e.Status)
}

// LoginRequest represents the request payload for login
type LoginRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	Code         string `json:"code,omitempty"`
	ResourceGUID string `json:"resourceGuid,omitempty"`
}

// LoginResponse represents the response from login
type LoginResponse struct {
	CodeRequested             bool `json:"codeRequested,omitempty"`
	EmailVerificationRequired bool `json:"emailVerificationRequired,omitempty"`
	UseSecurityKey            bool `json:"useSecurityKey,omitempty"`
	TwoFactorSetupRequired    bool `json:"twoFactorSetupRequired,omitempty"`
}

// User represents a user retrieved from the API
type User struct {
	UserID           string  `json:"userId"`
	Email            string  `json:"email"`
	Username         string  `json:"username"`
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	TwoFactorEnabled bool    `json:"twoFactorEnabled"`
	EmailVerified    bool    `json:"emailVerified"`
	ServerAdmin      bool    `json:"serverAdmin"`
	IDPName          *string `json:"idpName"`
	IDPID            *int `json:"idpId"`
}

// Org represents an organization
type Org struct {
	OrgID   string `json:"orgId"`
	Name    string `json:"name"`
	IsOwner *bool  `json:"isOwner,omitempty"`
}

// ListUserOrgsResponse represents the response from listing user organizations
type ListUserOrgsResponse struct {
	Orgs       []Org `json:"orgs"`
	Pagination struct {
		Total  int `json:"total"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"pagination"`
}

// DeviceWebAuthStartRequest represents the request payload for starting device web auth
type DeviceWebAuthStartRequest struct {
	ApplicationName string `json:"applicationName"`
	DeviceName      string `json:"deviceName,omitempty"`
}

// DeviceWebAuthStartResponse represents the response from starting device web auth
type DeviceWebAuthStartResponse struct {
	Code      string `json:"code"`
	ExpiresAt int64  `json:"expiresAt"` // Unix timestamp in milliseconds
}

// DeviceWebAuthPollResponse represents the response from polling device web auth
type DeviceWebAuthPollResponse struct {
	Verified bool   `json:"verified"`
	Token    string `json:"token,omitempty"` // Only present when verified is true
}

// CreateOlmRequest represents the request payload for creating an OLM
type CreateOlmRequest struct {
	Name string `json:"name"`
}

// CreateOlmResponse represents the response from creating an OLM
type CreateOlmResponse struct {
	OlmID  string `json:"olmId"`
	Secret string `json:"secret"`
}
