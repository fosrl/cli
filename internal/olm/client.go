package olm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	AgentName = "Pangolin CLI"
)

// Client handles communication with the OLM process via Unix socket
type Client struct {
	socketPath string
	httpClient *http.Client
}

// StatusError represents an error in the status response
type StatusError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// OLMStatusResponse represents the status response from OLM API
type StatusResponse struct {
	Connected       bool                   `json:"connected"`
	Registered      bool                   `json:"registered"`
	Terminated      bool                   `json:"terminated"`
	Version         string                 `json:"version,omitempty"`
	Agent           string                 `json:"agent,omitempty"`
	OrgID           string                 `json:"orgId,omitempty"`
	PeerStatuses    map[int]*OLMPeerStatus `json:"peers,omitempty"`
	NetworkSettings map[string]interface{} `json:"networkSettings,omitempty"`
	Error           *StatusError           `json:"error,omitempty"`
}

// OLMPeerStatus represents the status of a peer connection
type OLMPeerStatus struct {
	SiteID    int           `json:"siteId"`
	SiteName  string        `json:"name"`
	Connected bool          `json:"connected"`
	RTT       time.Duration `json:"rtt"`
	LastSeen  time.Time     `json:"lastSeen"`
	Endpoint  string        `json:"endpoint,omitempty"`
	IsRelay   bool          `json:"isRelay"`
	PeerIP    string        `json:"peerAddress,omitempty"`
}

// ExitResponse represents the exit/shutdown response
type ExitResponse struct {
	Status string `json:"status"`
}

// SwitchOrgRequest represents the switch org request
type SwitchOrgRequest struct {
	OrgID string `json:"org_id"`
}

// SwitchOrgResponse represents the switch org response
type SwitchOrgResponse struct {
	Status string `json:"status"`
}

// JITConnectionRequest represents a Just-In-Time connection request.
// Exactly one of SiteID or ResourceID must be set.
type JITConnectionRequest struct {
	Site     string `json:"site,omitempty"`
	Resource string `json:"resource,omitempty"`
}

// JITConnectionResponse represents the response from a JIT connection request
type JITConnectionResponse struct {
	Status string `json:"status"`
}

// NewClient creates a new OLM socket client
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = getDefaultSocketPath()
	}

	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{
			Timeout:   5 * time.Second,
			Transport: newHTTPTransport(socketPath),
		},
	}
}

// doRequest performs an HTTP request and handles common error cases
func (c *Client) doRequest(method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	return c.doRequestExpecting(method, path, body, headers, http.StatusOK)
}

// doRequestExpecting performs an HTTP request and treats the given status code as success
func (c *Client) doRequestExpecting(method, path string, body io.Reader, headers map[string]string, expectedStatus int) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://localhost"+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if !socketExists(c.socketPath) {
			return nil, fmt.Errorf("socket does not exist: %s (is the client running?)", c.socketPath)
		}
		return nil, fmt.Errorf("failed to connect to socket: %w", err)
	}

	if resp.StatusCode != expectedStatus {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// GetStatus retrieves the current status from the OLM process
func (c *Client) GetStatus() (*StatusResponse, error) {
	resp, err := c.doRequest("GET", "/status", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// Exit sends a shutdown signal to the OLM process
func (c *Client) Exit() (*ExitResponse, error) {
	resp, err := c.doRequest("POST", "/exit", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var exitResp ExitResponse
	if err := json.NewDecoder(resp.Body).Decode(&exitResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &exitResp, nil
}

// SwitchOrg switches to a different organization
func (c *Client) SwitchOrg(orgID string) (*SwitchOrgResponse, error) {
	reqBody := SwitchOrgRequest{OrgID: orgID}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/switch-org", bytes.NewBuffer(jsonData), map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var switchOrgResp SwitchOrgResponse
	if err := json.NewDecoder(resp.Body).Decode(&switchOrgResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &switchOrgResp, nil
}

// jitConnect is the shared implementation for JIT connection requests
func (c *Client) jitConnect(req JITConnectionRequest) (*JITConnectionResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequestExpecting("POST", "/jit-connect", bytes.NewBuffer(jsonData), map[string]string{
		"Content-Type": "application/json",
	}, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jitResp JITConnectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&jitResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &jitResp, nil
}

// JITConnectBySiteID initiates a dynamic Just-In-Time connection to the given site
func (c *Client) JITConnectBySiteID(siteID string) (*JITConnectionResponse, error) {
	if siteID == "" {
		return nil, fmt.Errorf("siteID must not be empty")
	}
	return c.jitConnect(JITConnectionRequest{Site: siteID})
}

// JITConnectByResourceID initiates a dynamic Just-In-Time connection to the site
// that serves the given resource
func (c *Client) JITConnectByResourceID(resourceID string) (*JITConnectionResponse, error) {
	if resourceID == "" {
		return nil, fmt.Errorf("resourceID must not be empty")
	}
	return c.jitConnect(JITConnectionRequest{Resource: resourceID})
}

// IsRunning checks if the OLM process is running by checking if the socket exists
// and making a health check request to verify the service is responding
func (c *Client) IsRunning() bool {
	// First check if socket exists
	if !socketExists(c.socketPath) {
		return false
	}

	// Then verify the service is actually responding by pinging /health
	resp, err := c.doRequest("GET", "/health", nil, nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return true
}
