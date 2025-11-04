package olm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultSocketPath = "/var/run/olm.sock"
)

// Client handles communication with the OLM process via Unix socket
type Client struct {
	socketPath string
	httpClient *http.Client
}

// StatusResponse represents the status response from OLM
type StatusResponse struct {
	Status     string          `json:"status"`
	Connected  bool            `json:"connected"`
	TunnelIP   string          `json:"tunnelIP"`
	Version    string          `json:"version"`
	Peers      map[string]Peer `json:"peers"`
	Registered bool            `json:"registered"` // whether the wireguard interface is created
	OrgID      string          `json:"orgId"`
}

// Peer represents a peer in the status response
type Peer struct {
	SiteID    int    `json:"siteId"`
	Connected bool   `json:"connected"`
	RTT       int64  `json:"rtt"` // nanoseconds
	LastSeen  string `json:"lastSeen"`
	Endpoint  string `json:"endpoint"`
	IsRelay   bool   `json:"isRelay"`
}

// ExitResponse represents the exit/shutdown response
type ExitResponse struct {
	Status string `json:"status"`
}

// SwitchOrgRequest represents the switch org request
type SwitchOrgRequest struct {
	OrgID string `json:"orgId"`
}

// SwitchOrgResponse represents the switch org response
type SwitchOrgResponse struct {
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
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", socketPath)
				},
			},
		},
	}
}

// getDefaultSocketPath returns the default socket path
// Checks config first, then falls back to default
func getDefaultSocketPath() string {
	if socketPath := viper.GetString("olm_defaults.socket_path"); socketPath != "" {
		return socketPath
	}
	return defaultSocketPath
}

// GetDefaultSocketPath returns the default socket path (exported for use in other packages)
func GetDefaultSocketPath() string {
	return getDefaultSocketPath()
}

// doRequest performs an HTTP request and handles common error cases
func (c *Client) doRequest(method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, "http://localhost"+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check if socket file exists
		if _, statErr := os.Stat(c.socketPath); os.IsNotExist(statErr) {
			return nil, fmt.Errorf("socket does not exist: %s (is the client running?)", c.socketPath)
		}
		return nil, fmt.Errorf("failed to connect to socket: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
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

	// print jsonData for debugging
	fmt.Printf("SwitchOrg request body: %s\n", string(jsonData))

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

// IsRunning checks if the OLM process is running by checking if the socket exists
func (c *Client) IsRunning() bool {
	_, err := os.Stat(c.socketPath)
	return err == nil
}
