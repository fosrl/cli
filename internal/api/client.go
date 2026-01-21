package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fosrl/cli/internal/version"
)

// ClientConfig holds configuration for creating a new client
type ClientConfig struct {
	BaseURL           string
	AgentName         string
	Token             string
	APIKey            string
	SessionCookieName string
	CSRFToken         string
}

// NewClient creates a new API client with the provided configuration
func NewClient(config ClientConfig) (*Client, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://app.pangolin.net"
	} else if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}

	// Default session cookie name
	sessionCookieName := config.SessionCookieName
	if sessionCookieName == "" {
		sessionCookieName = "p_session_token"
	}

	client := &Client{
		BaseURL:           strings.TrimSuffix(baseURL, "/"),
		AgentName:         config.AgentName,
		APIKey:            config.APIKey,
		Token:             config.Token,
		SessionCookieName: sessionCookieName,
		CSRFToken:         config.CSRFToken,
		HTTPClient: &HTTPClient{
			Timeout: 30 * time.Second,
		},
	}

	return client, nil
}

// Get performs a GET request to the API
func (c *Client) Get(endpoint string, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodGet, endpoint, nil, result, opts...)
}

// Post performs a POST request to the API
func (c *Client) Post(endpoint string, payload interface{}, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodPost, endpoint, payload, result, opts...)
}

// Put performs a PUT request to the API
func (c *Client) Put(endpoint string, payload interface{}, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodPut, endpoint, payload, result, opts...)
}

// Patch performs a PATCH request to the API
func (c *Client) Patch(endpoint string, payload interface{}, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodPatch, endpoint, payload, result, opts...)
}

// Delete performs a DELETE request to the API
func (c *Client) Delete(endpoint string, result interface{}, opts ...RequestOptions) error {
	return c.request(http.MethodDelete, endpoint, nil, result, opts...)
}

// request is the core method that handles all HTTP requests
func (c *Client) request(method, endpoint string, payload interface{}, result interface{}, opts ...RequestOptions) error {
	// Build URL
	requestURL, err := c.buildURL(endpoint, opts...)
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	// Prepare request body
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	// Create HTTP request
	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Set User-Agent with version
	userAgent := getUserAgent(c.AgentName)
	req.Header.Set("User-Agent", userAgent)

	// Set CSRF header if provided
	if c.CSRFToken != "" {
		req.Header.Set("X-CSRF-Token", c.CSRFToken)
	}

	// Set authentication
	if c.Token != "" {
		// Token is sent as a cookie
		cookie := &http.Cookie{
			Name:  c.SessionCookieName,
			Value: c.Token,
		}
		req.AddCookie(cookie)
	} else if c.APIKey != "" {
		// API key is sent as Bearer token
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	// Apply custom headers from options
	if len(opts) > 0 && opts[0].Headers != nil {
		for key, value := range opts[0].Headers {
			req.Header.Set(key, value)
		}
	}

	// Create HTTP client and execute request
	httpClient := &http.Client{
		Timeout: c.HTTPClient.Timeout,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if len(bodyBytes) == 0 {
		return nil
	}

	// Parse the API response structure
	var apiResp APIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check if the response indicates an error (based on error/success fields)
	if apiResp.Error.Bool() || !apiResp.Success {
		errorResp := ErrorResponse{
			Message: apiResp.Message,
			Status:  apiResp.Status,
			Stack:   apiResp.Stack,
		}
		// Use HTTP status code if status field is not set
		if errorResp.Status == 0 {
			errorResp.Status = resp.StatusCode
		}
		// If message is empty, try to provide a default based on status code
		if errorResp.Message == "" {
			switch errorResp.Status {
			case 401, 403:
				errorResp.Message = "Unauthorized"
			case 404:
				errorResp.Message = "Not found"
			case 500:
				errorResp.Message = "Internal server error"
			default:
				errorResp.Message = "An error occurred"
			}
		}
		return &errorResp
	}

	// Parse successful response
	if result != nil && apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return nil
}

// buildURL constructs the full URL for the request
func (c *Client) buildURL(endpoint string, opts ...RequestOptions) (string, error) {
	// Ensure endpoint starts with /
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	baseURL := strings.TrimSuffix(c.BaseURL, "/")
	fullURL := baseURL + endpoint

	// Add query parameters if provided
	if len(opts) > 0 && opts[0].Query != nil && len(opts[0].Query) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return "", err
		}

		q := u.Query()
		for key, value := range opts[0].Query {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	return fullURL, nil
}

// SetBaseURL updates the base URL for the client
func (c *Client) SetBaseURL(baseURL string) {
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}
	c.BaseURL = strings.TrimSuffix(baseURL, "/")
}

// SetToken updates the token for the client
func (c *Client) SetToken(token string) {
	c.Token = token
}

// Logout logs out the current user
func (c *Client) Logout() error {
	var result EmptyResponse
	err := c.Post("/auth/logout", nil, &result)
	if err != nil {
		return err
	}
	return nil
}

// GetUser retrieves the current user information
func (c *Client) GetUser() (*User, error) {
	var user User
	err := c.Get("/user", &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetServerInfo retrieves server information including version, build type, and license status
func (c *Client) GetServerInfo() (*ServerInfo, error) {
	var serverInfo ServerInfo
	err := c.Get("/server-info", &serverInfo)
	if err != nil {
		return nil, err
	}
	return &serverInfo, nil
}

// ListUserOrgs lists organizations for a user
func (c *Client) ListUserOrgs(userID string) (*ListUserOrgsResponse, error) {
	path := fmt.Sprintf("/user/%s/orgs", userID)
	var response ListUserOrgsResponse
	err := c.Get(path, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// CreateOlm creates an OLM for a user
func (c *Client) CreateOlm(userID, name string) (*CreateOlmResponse, error) {
	requestBody := CreateOlmRequest{
		Name: name,
	}
	path := fmt.Sprintf("/user/%s/olm", userID)
	var response CreateOlmResponse
	err := c.Put(path, requestBody, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// GetUserOlm gets an OLM for a user by userId and olmId
// If orgID is provided, it will be passed as a query parameter
func (c *Client) GetUserOlm(userID, olmID string, orgID ...string) (*Olm, error) {
	path := fmt.Sprintf("/user/%s/olm/%s", userID, olmID)
	var olm Olm

	var opts []RequestOptions
	if len(orgID) > 0 && orgID[0] != "" {
		opts = []RequestOptions{
			{
				Query: map[string]string{
					"orgId": orgID[0],
				},
			},
		}
	}

	err := c.Get(path, &olm, opts...)
	if err != nil {
		return nil, err
	}
	return &olm, nil
}

// RecoverOlm attempts to recover an existing olm's secret
// based on the device's platform fingerprint. This is useful
// for device reinstalls so that a new device isn't always
// created on the server side.
func (c *Client) RecoverOlmFromFingerprint(userID string, platformFingerprint string) (*RecoverOlmResponse, error) {
	requestBody := RecoverOlmRequest{
		PlatformFingerprint: platformFingerprint,
	}

	path := fmt.Sprintf("/user/%s/olm/recover", userID)
	var response RecoverOlmResponse
	err := c.Post(path, requestBody, &response)
	if err != nil {
		// print the error
		fmt.Printf("Error recovering olm: %v\n", err)
		return nil, err
	}

	return &response, nil
}

// GetOrg gets an organization by ID
func (c *Client) GetOrg(orgID string) (*GetOrgResponse, error) {
	path := fmt.Sprintf("/org/%s", orgID)
	var response GetOrgResponse
	err := c.Get(path, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// CheckOrgUserAccess checks if a user has access to an organization
func (c *Client) CheckOrgUserAccess(orgID, userID string) (*CheckOrgUserAccessResponse, error) {
	path := fmt.Sprintf("/org/%s/user/%s/check", orgID, userID)
	var response CheckOrgUserAccessResponse
	err := c.Get(path, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// GetClient gets a client by ID
func (c *Client) GetClient(clientID int) (*GetClientResponse, error) {
	path := fmt.Sprintf("/client/%d", clientID)
	var response GetClientResponse
	err := c.Get(path, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// GetMyDevice gets the current device information including user, organizations, and OLM
func (c *Client) GetMyDevice(olmID string) (*MyDeviceResponse, error) {
	// Build query parameters
	params := url.Values{}
	params.Set("olmId", olmID)
	path := fmt.Sprintf("/my-device?%s", params.Encode())
	var response MyDeviceResponse
	err := c.Get(path, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// TestConnection tests the connection to the API server
func (c *Client) TestConnection() (bool, error) {
	// Create a temporary client with shorter timeout for connection test
	testClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Use HEAD request to test connection
	fullURL := c.BaseURL
	req, err := http.NewRequest("HEAD", fullURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := getUserAgent(c.AgentName)
	req.Header.Set("User-Agent", userAgent)

	resp, err := testClient.Do(req)
	if err != nil {
		return false, nil // Return false (not an error) if connection fails
	}
	defer resp.Body.Close()

	// Consider 200-299 and 404 as successful connection
	return (resp.StatusCode >= 200 && resp.StatusCode < 300) || resp.StatusCode == 404, nil
}

// CheckHealth checks if the server is reachable and responding
// Returns true if status is 200-299, 401, or 403 (server is up)
// Returns false with error if server is unreachable or returns other status codes
func (c *Client) CheckHealth() (bool, error) {
	// Create a temporary client with shorter timeout for health check
	testClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Use GET request to root endpoint
	fullURL := c.BaseURL
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := getUserAgent(c.AgentName)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	// Set authentication if available
	if c.Token != "" {
		cookie := &http.Cookie{
			Name:  c.SessionCookieName,
			Value: c.Token,
		}
		req.AddCookie(cookie)
	} else if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := testClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("server unreachable: %w", err)
	}
	defer resp.Body.Close()

	// Return true for 200-299, 401, or 403 (server is up)
	// Return false for other status codes (server returned an error)
	if (resp.StatusCode >= 200 && resp.StatusCode < 300) || resp.StatusCode == 401 || resp.StatusCode == 403 {
		return true, nil
	}

	return false, fmt.Errorf("server returned status %d", resp.StatusCode)
}

// Helper functions for API calls

// normalizeBaseURL normalizes a base URL by adding protocol if missing and trimming trailing slashes
func normalizeBaseURL(baseURL string) string {
	if baseURL == "" {
		baseURL = "https://app.pangolin.net"
	}
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}
	return strings.TrimSuffix(baseURL, "/")
}

// buildAPIBaseURL builds the API v1 base URL, ensuring it ends with /api/v1
func buildAPIBaseURL(baseURL string) string {
	baseURL = normalizeBaseURL(baseURL)

	// Ensure we're using the API v1 endpoint
	if !strings.Contains(baseURL, "/api/v1") {
		baseURL = baseURL + "/api/v1"
	} else if !strings.HasSuffix(baseURL, "/api/v1") {
		// If it contains /api/v1 but not at the end, trim any trailing path after /api/v1
		idx := strings.Index(baseURL, "/api/v1")
		if idx != -1 {
			baseURL = baseURL[:idx+7] // Keep up to and including "/api/v1"
		}
	}

	return baseURL
}

// getUserAgent returns the user agent string formatted as "pangolin-cli-version"
func getUserAgent(agentName string) string {
	return fmt.Sprintf("pangolin-cli-%s", version.Version)
}

// setJSONRequestHeaders sets common headers for JSON API requests
func setJSONRequestHeaders(req *http.Request, userAgent string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
}

// setJSONResponseHeaders sets headers for JSON API requests (without Content-Type for GET requests)
func setJSONResponseHeaders(req *http.Request, userAgent string) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
}

// createHTTPClient creates an HTTP client with the specified timeout
func createHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

// parseAPIResponseBody parses the response body into an APIResponse struct
func parseAPIResponseBody(bodyBytes []byte) (*APIResponse, error) {
	var apiResp APIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &apiResp, nil
}

// createErrorResponse creates an ErrorResponse from an APIResponse and HTTP status code
func createErrorResponse(apiResp *APIResponse, httpStatusCode int, getDefaultMessage func(int) string) *ErrorResponse {
	errorResp := ErrorResponse{
		Message: apiResp.Message,
		Status:  apiResp.Status,
		Stack:   apiResp.Stack,
	}

	if errorResp.Status == 0 {
		errorResp.Status = httpStatusCode
	}

	if errorResp.Message == "" && getDefaultMessage != nil {
		errorResp.Message = getDefaultMessage(errorResp.Status)
	}

	return &errorResp
}

// getDefaultErrorMessage returns a default error message based on status code
func getDefaultErrorMessage(statusCode int) string {
	switch statusCode {
	case 400:
		return "Bad request"
	case 401, 403:
		return "Unauthorized"
	case 404:
		return "Not found"
	case 429:
		return "Rate limit exceeded"
	case 500:
		return "Internal server error"
	default:
		return "An error occurred"
	}
}

// getDeviceAuthErrorMessage returns a default error message for device auth endpoints
func getDeviceAuthErrorMessage(statusCode int) string {
	switch statusCode {
	case 400:
		return "Bad request"
	case 403:
		return "IP address mismatch"
	case 429:
		return "Rate limit exceeded"
	case 500:
		return "Internal server error"
	default:
		return "An error occurred"
	}
}

// LoginWithCookie performs a login request and returns the session cookie
// This is a lower-level function that handles cookie extraction
func LoginWithCookie(client *Client, req LoginRequest) (*LoginResponse, string, error) {
	var response LoginResponse
	sessionToken := ""

	// Build URL
	baseURL := normalizeBaseURL(client.BaseURL)
	endpoint := "/api/v1/auth/login"
	url := baseURL + endpoint

	// Marshal request body
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	userAgent := getUserAgent(client.AgentName)
	setJSONRequestHeaders(httpReq, userAgent)

	// Set CSRF token header
	csrfToken := client.CSRFToken
	if csrfToken == "" {
		csrfToken = "x-csrf-protection"
	}
	httpReq.Header.Set("X-CSRF-Token", csrfToken)

	// Execute request
	httpClient := createHTTPClient(client.HTTPClient.Timeout)
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Extract session cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == client.SessionCookieName || cookie.Name == "p_session" {
			sessionToken = cookie.Value
			break
		}
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	if len(bodyBytes) == 0 {
		// Return empty response but with token if available
		return &response, sessionToken, nil
	}

	// Parse the API response
	apiResp, err := parseAPIResponseBody(bodyBytes)
	if err != nil {
		return nil, "", err
	}

	// Check if the response indicates an error
	if apiResp.Error.Bool() || !apiResp.Success {
		// Try to extract message from raw response if it exists in a different format
		if apiResp.Message == "" {
			var rawResp map[string]interface{}
			if json.Unmarshal(bodyBytes, &rawResp) == nil {
				if msg, ok := rawResp["message"].(string); ok && msg != "" {
					apiResp.Message = msg
				}
			}
		}

		errorResp := createErrorResponse(apiResp, resp.StatusCode, getDefaultErrorMessage)
		return nil, "", errorResp
	}

	// Parse successful response data
	if apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, &response); err != nil {
			return nil, "", fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return &response, sessionToken, nil
}

// StartDeviceWebAuth requests a device code from the server
// The client should have BaseURL set but no authentication token is required
func StartDeviceWebAuth(client *Client, req DeviceWebAuthStartRequest) (*DeviceWebAuthStartResponse, error) {
	var response DeviceWebAuthStartResponse

	// Build URL
	baseURL := buildAPIBaseURL(client.BaseURL)
	endpoint := "/auth/device-web-auth/start"
	url := baseURL + endpoint

	// Marshal request body
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	userAgent := getUserAgent(client.AgentName)
	setJSONRequestHeaders(httpReq, userAgent)

	// Set CSRF token header
	csrfToken := client.CSRFToken
	if csrfToken == "" {
		csrfToken = "x-csrf-protection"
	}
	httpReq.Header.Set("X-CSRF-Token", csrfToken)

	// Execute request
	httpClient := createHTTPClient(client.HTTPClient.Timeout)
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the API response
	apiResp, err := parseAPIResponseBody(bodyBytes)
	if err != nil {
		return nil, err
	}

	// Check if the response indicates an error
	if apiResp.Error.Bool() || !apiResp.Success {
		errorResp := createErrorResponse(apiResp, resp.StatusCode, getDeviceAuthErrorMessage)
		return nil, errorResp
	}

	// Parse successful response data
	if apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return &response, nil
}

// PollDeviceWebAuth polls the server to check if the device code has been verified
// The client should have BaseURL set but no authentication token is required
func PollDeviceWebAuth(client *Client, code string) (*DeviceWebAuthPollResponse, string, error) {
	var response DeviceWebAuthPollResponse

	// Build URL
	baseURL := buildAPIBaseURL(client.BaseURL)
	endpoint := fmt.Sprintf("/auth/device-web-auth/poll/%s", code)
	url := baseURL + endpoint

	// Create request
	httpReq, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	userAgent := getUserAgent(client.AgentName)
	setJSONResponseHeaders(httpReq, userAgent)

	// Set CSRF token header
	csrfToken := client.CSRFToken
	if csrfToken == "" {
		csrfToken = "x-csrf-protection"
	}
	httpReq.Header.Set("X-CSRF-Token", csrfToken)

	// Execute request
	httpClient := createHTTPClient(client.HTTPClient.Timeout)
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the API response
	apiResp, err := parseAPIResponseBody(bodyBytes)
	if err != nil {
		return nil, "", err
	}

	message := apiResp.Message

	// Check if the response indicates an error
	if apiResp.Error.Bool() || !apiResp.Success {
		errorResp := createErrorResponse(apiResp, resp.StatusCode, getDeviceAuthErrorMessage)
		return nil, message, errorResp
	}

	// Parse successful response data
	if apiResp.Data != nil {
		if err := json.Unmarshal(apiResp.Data, &response); err != nil {
			return nil, message, fmt.Errorf("failed to unmarshal response data: %w", err)
		}
	}

	return &response, message, nil
}

func (c *Client) ApplyBlueprint(orgID string, name string, blueprint string) (*ApplyBlueprintResponse, error) {
	// Create request payload with raw YAML content
	requestBody := ApplyBlueprintRequest{
		Name:      name,
		Blueprint: blueprint,
	}

	path := fmt.Sprintf("/org/%s/blueprint", orgID)
	var response ApplyBlueprintResponse
	err := c.Put(path, requestBody, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
