package login

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/utils"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

type HostingOption string

const (
	HostingOptionCloud      HostingOption = "cloud"
	HostingOptionSelfHosted HostingOption = "self-hosted"
)

// getDeviceName returns a human-readable device name
func getDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "Unknown Device"
	}
	return hostname
}

func loginWithWeb(hostname string) (string, error) {
	// Build base URL for login (use hostname as-is, StartDeviceWebAuth will add /api/v1)
	baseURL := hostname

	// Create a temporary API client for login (without auth)
	loginClient, err := api.NewClient(api.ClientConfig{
		BaseURL:           baseURL,
		AgentName:         "pangolin-cli",
		SessionCookieName: "p_session_token",
		CSRFToken:         "x-csrf-protection",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create API client: %w", err)
	}

	// Get device name
	deviceName := getDeviceName()

	// Request device code
	startReq := api.DeviceWebAuthStartRequest{
		ApplicationName: "Pangolin CLI",
		DeviceName:      deviceName,
	}

	startResp, err := api.StartDeviceWebAuth(loginClient, startReq)
	if err != nil {
		return "", fmt.Errorf("failed to start device web auth: %w", err)
	}

	code := startResp.Code
	// Calculate expiry time from relative seconds
	expiresAt := time.Now().Add(time.Duration(startResp.ExpiresInSeconds) * time.Second)

	// Build the base login URL (without query parameter) for display
	baseLoginURL := fmt.Sprintf("%s/auth/login/device", strings.TrimSuffix(hostname, "/"))
	// Build the login URL with code as query parameter for browser
	loginURL := fmt.Sprintf("%s?code=%s", baseLoginURL, code)

	// Display code and instructions (similar to GH CLI format)
	logger.Info("First copy your one-time code: %s", code)
	logger.Info("Press Enter to open %s in your browser...", baseLoginURL)

	// Wait for Enter in a goroutine (non-blocking) and open browser when pressed
	go func() {
		reader := bufio.NewReader(os.Stdin)
		_, err := reader.ReadString('\n')
		if err == nil {
			// User pressed Enter, open browser
			if err := browser.OpenURL(loginURL); err != nil {
				// Don't fail if browser can't be opened, just warn
				logger.Warning("Failed to open browser automatically")
				logger.Info("Please manually visit: %s", baseLoginURL)
			}
		}
	}()

	// Poll for verification (starts immediately, doesn't wait for Enter)
	pollInterval := 1 * time.Second
	startTime := time.Now()
	maxPollDuration := 5 * time.Minute

	var token string

	for {
		// print
		logger.Debug("Polling for device web auth verification...")
		// Check if code has expired
		if time.Now().After(expiresAt) {
			logger.Error("Device web auth code has expired")
			return "", fmt.Errorf("code expired. Please try again")
		}

		// Check if we've exceeded max polling duration
		if time.Since(startTime) > maxPollDuration {
			logger.Error("Polling timed out after %v", maxPollDuration)
			return "", fmt.Errorf("polling timeout. Please try again")
		}

		// Poll for verification status
		pollResp, message, err := api.PollDeviceWebAuth(loginClient, code)
		// print debug info
		logger.Debug("Polling response: %+v, message: %s, err: %v", pollResp, message, err)
		if err != nil {
			logger.Error("Error polling device web auth: %v", err)
			return "", fmt.Errorf("failed to poll device web auth: %w", err)
		}

		// Check verification status
		if pollResp.Verified {
			token = pollResp.Token
			if token == "" {
				logger.Error("Verification succeeded but no token received")
				return "", fmt.Errorf("verification succeeded but no token received")
			}
			return token, nil
		}

		// Check for expired or not found messages
		if message == "Code expired" || message == "Code not found" {
			logger.Error("Device web auth code has expired or not found")
			return "", fmt.Errorf("code expired or not found. Please try again")
		}

		// Wait before next poll
		time.Sleep(pollInterval)
	}
}

var LoginCmd = &cobra.Command{
	Use:   "login [hostname]",
	Short: "Login to Pangolin",
	Long:  "Interactive login to select your hosting option and configure access.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiClient := api.FromContext(cmd.Context())
		accountStore := config.AccountStoreFromContext(cmd.Context())

		var hostingOption HostingOption
		var hostname string

		// Check if hostname was provided as positional argument
		if len(args) > 0 {
			hostname = args[0]
		}

		// If hostname was provided, skip hosting option selection
		if hostname == "" {
			// First question: select hosting option
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[HostingOption]().
						Title("Select your hosting option").
						Options(
							huh.NewOption("Pangolin Cloud (app.pangolin.net)", HostingOptionCloud),
							huh.NewOption("Self-hosted or Dedicated instance", HostingOptionSelfHosted),
						).
						Value(&hostingOption),
				),
			)

			if err := form.Run(); err != nil {
				logger.Error("Error: %v", err)
				return
			}

			// If self-hosted, prompt for hostname
			if hostingOption == HostingOptionSelfHosted {
				hostnameForm := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Enter hostname URL").
							Placeholder("https://your-instance.example.com").
							Value(&hostname),
					),
				)

				if err := hostnameForm.Run(); err != nil {
					logger.Error("Error: %v", err)
					return
				}
			} else {
				// For cloud, set the default hostname
				hostname = "app.pangolin.net"
			}
		}

		// Normalize hostname (preserve protocol, remove trailing slash)
		hostname = strings.TrimSuffix(hostname, "/")

		// If no protocol specified, default to https
		if !strings.HasPrefix(hostname, "http://") && !strings.HasPrefix(hostname, "https://") {
			hostname = "https://" + hostname
		}

		// Perform web login
		sessionToken, err := loginWithWeb(hostname)
		if err != nil {
			logger.Error("%v", err)
			return
		}

		if sessionToken == "" {
			logger.Error("Login appeared successful but no session token was received.")
			return
		}

		// Update the global API client (always initialized)
		// Update base URL and token (hostname already includes protocol)
		apiBaseURL := hostname + "/api/v1"
		apiClient.SetBaseURL(apiBaseURL)
		apiClient.SetToken(sessionToken)

		logger.Success("Device authorized")
		fmt.Println()

		// Get user information
		var user *api.User
		user, err = apiClient.GetUser()
		if err != nil {
			logger.Error("Failed to get user information: %v", err)
			return // FIXME: handle errors properly with exit codes!
		}

		if _, exists := accountStore.Accounts[user.UserID]; exists {
			logger.Warning("Already logged in as this user; no action needed")
			return
		}

		// Ensure OLM credentials exist and are valid
		userID := user.UserID

		orgID, err := utils.SelectOrgForm(apiClient, userID)
		if err != nil {
			logger.Error("Failed to select organization: %v", err)
			return
		}

		newOlmCreds, err := apiClient.CreateOlm(userID, utils.GetDeviceName())
		if err != nil {
			logger.Error("Failed to obtain olm credentials: %v", err)
			return
		}

		newAccount := config.Account{
			UserID:       userID,
			Host:         hostname,
			Email:        user.Email,
			SessionToken: sessionToken,
			OrgID:        orgID,
			OlmCredentials: &config.OlmCredentials{
				ID:     newOlmCreds.OlmID,
				Secret: newOlmCreds.Secret,
			},
		}

		accountStore.Accounts[user.UserID] = newAccount
		accountStore.ActiveUserID = userID

		err = accountStore.Save()
		if err != nil {
			logger.Error("Failed to save account store: %s", err)
			logger.Warning("You may not be able to login properly until this is saved.")
			return
		}

		// List and select organization
		if user != nil {
			if _, err := utils.SelectOrgForm(apiClient, user.UserID); err != nil {
				logger.Warning("%v", err)
			}
		}

		// Print logged in message after all setup is complete
		if user != nil {
			displayName := user.Email
			if displayName == "" && user.Username != nil && *user.Username != "" {
				displayName = *user.Username
			}
			if displayName != "" {
				logger.Success("Logged in as %s", displayName)
			}
		}
	},
}
