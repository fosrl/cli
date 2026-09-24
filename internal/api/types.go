package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Client represents the API client configuration
type Client struct {
	BaseURL    string
	AgentName  string
	Session    ClientSession
	HTTPClient *HTTPClient
}

// HTTPClient wraps the standard http.Client with additional configuration
type HTTPClient struct {
	Timeout       time.Duration
	TLSClientCert string
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
	Id       string  `json:"id"`
	UserID   string  `json:"userId"` // Alias for Id, used in some contexts
	Email    string  `json:"email"`
	Username *string `json:"username,omitempty"`
	Name     *string `json:"name,omitempty"`
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
	Code             string `json:"code"`
	ExpiresInSeconds int64  `json:"expiresInSeconds"` // Relative seconds until expiry
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
	ID     string `json:"id"`
	OlmID  string `json:"olmId"`
	Secret string `json:"secret"`
	Name   string `json:"name"`
}

type RecoverOlmRequest struct {
	PlatformFingerprint string `json:"platformFingerprint"`
}

type RecoverOlmResponse struct {
	OlmID  string `json:"olmId"`
	Secret string `json:"secret"`
}

// EmptyResponse represents an empty API response
type EmptyResponse struct{}

// GetOrgResponse represents the response for getting an organization
type GetOrgResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CheckOrgUserAccessResponse represents the response for checking org user access
type CheckOrgUserAccessResponse struct {
	Allowed  bool         `json:"allowed"`
	Error    *string      `json:"error,omitempty"`
	Policies *OrgPolicies `json:"policies,omitempty"`
}

// OrgPolicies represents organization policies
type OrgPolicies struct {
	RequiredTwoFactor *bool             `json:"requiredTwoFactor,omitempty"`
	MaxSessionLength  *MaxSessionLength `json:"maxSessionLength,omitempty"`
	PasswordAge       *PasswordAge      `json:"passwordAge,omitempty"`
}

// MaxSessionLength represents max session length policy
type MaxSessionLength struct {
	Compliant             bool    `json:"compliant"`
	MaxSessionLengthHours int     `json:"maxSessionLengthHours"`
	SessionAgeHours       float64 `json:"sessionAgeHours"`
}

// PasswordAge represents password age policy
type PasswordAge struct {
	Compliant          bool    `json:"compliant"`
	MaxPasswordAgeDays int     `json:"maxPasswordAgeDays"`
	PasswordAgeDays    float64 `json:"passwordAgeDays"`
}

// GetClientResponse represents the response for getting a client
type GetClientResponse struct {
	Id    int     `json:"id"`
	Name  string  `json:"name"`
	OlmID *string `json:"olmId,omitempty"`
}

// MyDeviceUser represents a user in the my device response
type MyDeviceUser struct {
	UserID           string  `json:"userId"`
	Email            string  `json:"email"`
	Username         *string `json:"username,omitempty"`
	Name             *string `json:"name,omitempty"`
	Type             *string `json:"type,omitempty"`
	TwoFactorEnabled *bool   `json:"twoFactorEnabled,omitempty"`
	EmailVerified    *bool   `json:"emailVerified,omitempty"`
	ServerAdmin      *bool   `json:"serverAdmin,omitempty"`
	IDPName          *string `json:"idpName,omitempty"`
	IDPID            *int    `json:"idpId,omitempty"`
}

// ResponseOrg represents an organization in the my device response
type ResponseOrg struct {
	OrgID   string `json:"orgId"`
	OrgName string `json:"orgName"`
	RoleID  int    `json:"roleId"`
}

// Olm represents an OLM (Online Management) record
type Olm struct {
	OlmID   string  `json:"olmId"`
	UserID  string  `json:"userId"`
	Name    *string `json:"name,omitempty"`
	Secret  *string `json:"secret,omitempty"`
	Blocked *bool   `json:"blocked,omitempty"` // Indicates if the OLM is blocked
}

// MyDeviceResponse represents the response for getting my device
type MyDeviceResponse struct {
	User MyDeviceUser  `json:"user"`
	Orgs []ResponseOrg `json:"orgs"`
	Olm  *Olm          `json:"olm,omitempty"`
}

// ServerInfo represents server information including version, build type, and license status
type ServerInfo struct {
	Version                string  `json:"version"`
	SupporterStatusValid   bool    `json:"supporterStatusValid"`
	Build                  string  `json:"build"` // "oss" | "enterprise" | "saas"
	EnterpriseLicenseValid bool    `json:"enterpriseLicenseValid"`
	EnterpriseLicenseType  *string `json:"enterpriseLicenseType,omitempty"`
}

// ApplyBlueprintRequest represents a new blueprint application request.
type ApplyBlueprintRequest struct {
	Name      string `json:"name"`
	Blueprint string `json:"blueprint"`
	Source    string `json:"source,omitempty"`
}

type ApplyBlueprintResponse struct {
	Name        string  `json:"name"`
	OrgID       string  `json:"orgId"`
	Source      string  `json:"source"`
	Message     *string `json:"message"`
	BlueprintID int     `json:"blueprintId"`
	Succeeded   bool    `json:"succeeded"`
	Contents    string  `json:"contents"`
}

type SignSSHKeyRequest struct {
	PublicKey string `json:"publicKey"`
	Resource  string `json:"resource"`
	Username  string `json:"username,omitempty"`
}

type SignSSHKeyData struct {
	MessageIDs       []int64  `json:"messageIds"`
	MessageID        int64    `json:"messageId"`
	Certificate      string   `json:"certificate"`
	KeyID            string   `json:"keyId"`
	ValidPrincipals  []string `json:"validPrincipals"`
	ValidAfter       string   `json:"validAfter"`
	ValidBefore      string   `json:"validBefore"`
	ExpiresInSeconds int      `json:"expiresIn"`
	AuthDaemonMode   string   `json:"authDaemonMode"` // "internal" or "agent"
	Hostname         string   `json:"sshHost"`        // hostname for SSH connection (returned by API)
	User             string   `json:"sshUsername"`    // user for SSH connection (returned by API)
	ResourceID       int      `json:"resourceId"`     // resource ID for SSH connection (returned by API)
	SiteIDs          []int    `json:"siteIds"`        // site ID for SSH connection (returned by API)
	SiteID           int      `json:"siteId"`         // site ID for SSH connection (returned by API)
}

type RoundTripMessage struct {
	MessageID  int64   `json:"messageId"`
	Complete   bool    `json:"complete"`
	SentAt     int64   `json:"sentAt"`     // epoch seconds
	ReceivedAt int64   `json:"receivedAt"` // epoch seconds
	Error      *string `json:"error,omitempty"`
}

type SignSSHKeyResponse struct {
	Success bool           `json:"success"`
	Error   *string        `json:"error,omitempty"`
	Data    SignSSHKeyData `json:"data"`
}

// UserResourceAliasItem is one alias with optional label names from includeLabels=true.
type UserResourceAliasItem struct {
	Alias  string   `json:"alias"`
	Labels []string `json:"labels"`
}

// ListUserResourceAliasesData is the inner `data` of GET /org/:orgId/user-resource-aliases.
type ListUserResourceAliasesData struct {
	Aliases    []string                `json:"aliases"`
	Items      []UserResourceAliasItem `json:"items,omitempty"`
	Pagination AliasesPagination       `json:"pagination"`
}

// AliasesPagination matches the paginated API envelope for user-resource-aliases.
type AliasesPagination struct {
	Total    int `json:"total"`
	PageSize int `json:"pageSize"`
	Page     int `json:"page"`
}

// LauncherLabel is a label attached to a launcher resource.
type LauncherLabel struct {
	LabelID int    `json:"labelId"`
	Name    string `json:"name"`
	Color   string `json:"color"`
}

// LauncherSiteInfo is the site a private launcher resource belongs to.
type LauncherSiteInfo struct {
	SiteID int    `json:"siteId"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Online *bool  `json:"online,omitempty"`
}

// LauncherResource mirrors server/routers/launcher/types.ts's LauncherResource: a
// unified view of both public resources and private site resources, with the
// access URL already computed server-side.
type LauncherResource struct {
	LauncherResourceKey string            `json:"launcherResourceKey"`
	ResourceType        string            `json:"resourceType"` // "public" | "site"
	ResourceID          int               `json:"resourceId"`
	SiteResourceID      *int              `json:"siteResourceId,omitempty"`
	NiceID              string            `json:"niceId"`
	Name                string            `json:"name"`
	AccessDisplay       string            `json:"accessDisplay"`
	AccessCopyValue     string            `json:"accessCopyValue"`
	AccessURL           *string           `json:"accessUrl"`
	IconURL             *string           `json:"iconUrl"`
	Enabled             bool              `json:"enabled"`
	Mode                string            `json:"mode"`
	Labels              []LauncherLabel   `json:"labels"`
	Site                *LauncherSiteInfo `json:"site,omitempty"`
}

// LauncherPagination matches the paginated API envelope for launcher resources.
type LauncherPagination struct {
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// ListLauncherResourcesData is the inner `data` of GET /org/:orgId/launcher/resources.
type ListLauncherResourcesData struct {
	Resources  []LauncherResource `json:"resources"`
	Pagination LauncherPagination `json:"pagination"`
}

// GetResourceData is the (partial) inner `data` of GET /org/:orgId/resource/:niceId.
// Only the fields the CLI needs are modeled; the server returns more, which
// json.Unmarshal simply ignores.
type GetResourceData struct {
	ResourceID   int    `json:"resourceId"`
	ResourceGUID string `json:"resourceGuid"`
	OrgID        string `json:"orgId"`
	Name         string `json:"name"`
	NiceID       string `json:"niceId"`
	Mode         string `json:"mode"`
}

// VirtualApiKeySummary is the public (secret-free) shape of a virtual API key.
type VirtualApiKeySummary struct {
	VirtualApiKeyID string `json:"virtualApiKeyId"`
	OrgID           string `json:"orgId"`
	Name            string `json:"name"`
	LastChars       string `json:"lastChars"`
	Kind            string `json:"kind"`
	CreatedAt       int64  `json:"createdAt"`
}

// ListMyVirtualApiKeysData is the inner `data` of GET /org/:orgId/my-virtual-api-keys.
type ListMyVirtualApiKeysData struct {
	UserKey      VirtualApiKeySummary   `json:"userKey"`
	ManualKeys   []VirtualApiKeySummary `json:"manualKeys"`
	ResourceName *string                `json:"resourceName,omitempty"`
}

// GetMyVirtualApiKeyData is the inner `data` of GET /org/:orgId/my-virtual-api-keys/:virtualApiKeyId.
type GetMyVirtualApiKeyData struct {
	VirtualApiKey struct {
		VirtualApiKeyID string `json:"virtualApiKeyId"`
		Secret          string `json:"secret"`
		LastChars       string `json:"lastChars"`
	} `json:"virtualApiKey"`
}
