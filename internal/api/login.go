package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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

// getUserAgent returns the user agent string, defaulting to "pangolin-cli" if empty
func getUserAgent(agentName string) string {
	if agentName == "" {
		return "pangolin-cli"
	}
	return agentName
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

// Logout performs a logout request
func (c *Client) Logout() error {
	var result interface{}
	err := c.Post("/auth/logout", nil, &result)
	if err != nil {
		return err
	}
	return nil
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
