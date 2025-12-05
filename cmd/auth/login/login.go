package login

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/secrets"
	"github.com/fosrl/cli/internal/utils"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	utils.Info("First copy your one-time code: %s", code)
	utils.Info("Press Enter to open %s in your browser...", baseLoginURL)

	// Wait for Enter in a goroutine (non-blocking) and open browser when pressed
	go func() {
		reader := bufio.NewReader(os.Stdin)
		_, err := reader.ReadString('\n')
		if err == nil {
			// User pressed Enter, open browser
			if err := browser.OpenURL(loginURL); err != nil {
				// Don't fail if browser can't be opened, just warn
				utils.Warning("Failed to open browser automatically")
				utils.Info("Please manually visit: %s", baseLoginURL)
			}
		}
	}()

	// Poll for verification (starts immediately, doesn't wait for Enter)
	pollInterval := 1 * time.Second
	startTime := time.Now()
	maxPollDuration := 5 * time.Minute

	var token string

	for {
		//print
		utils.Debug("Polling for device web auth verification...")
		// Check if code has expired
		if time.Now().After(expiresAt) {
			utils.Error("Device web auth code has expired")
			return "", fmt.Errorf("code expired. Please try again")
		}

		// Check if we've exceeded max polling duration
		if time.Since(startTime) > maxPollDuration {
			utils.Error("Polling timed out after %v", maxPollDuration)
			return "", fmt.Errorf("polling timeout. Please try again")
		}

		// Poll for verification status
		pollResp, message, err := api.PollDeviceWebAuth(loginClient, code)
		// print debug info
		utils.Debug("Polling response: %+v, message: %s, err: %v", pollResp, message, err)
		if err != nil {
			utils.Error("Error polling device web auth: %v", err)
			return "", fmt.Errorf("failed to poll device web auth: %w", err)
		}

		// Check verification status
		if pollResp.Verified {
			token = pollResp.Token
			if token == "" {
				utils.Error("Verification succeeded but no token received")
				return "", fmt.Errorf("verification succeeded but no token received")
			}
			return token, nil
		}

		// Check for expired or not found messages
		if message == "Code expired" || message == "Code not found" {
			utils.Error("Device web auth code has expired or not found")
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
		// Check if user is already logged in
		if err := utils.EnsureLoggedIn(); err == nil {
			// User is logged in, show error with account info
			email := viper.GetString("email")
			var accountInfo string
			if email != "" {
				accountInfo = fmt.Sprintf(" (%s)", email)
			}
			utils.Error("You are already logged in%s. Please logout first using 'pangolin logout'", accountInfo)
			return
		}

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
				utils.Error("Error: %v", err)
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
					utils.Error("Error: %v", err)
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

		// Store hostname in viper config (with protocol)
		viper.Set("hostname", hostname)

		// Ensure config type is set and file path is correct
		if viper.ConfigFileUsed() == "" {
			// Config file doesn't exist yet, set the full path
			// Get .pangolin directory and ensure it exists
			pangolinDir, err := utils.GetPangolinDir()
			if err == nil {
				viper.SetConfigFile(filepath.Join(pangolinDir, "pangolin.json"))
				viper.SetConfigType("json")
			}
		}

		if err := viper.WriteConfig(); err != nil {
			// If config file doesn't exist, create it
			if err := viper.SafeWriteConfig(); err != nil {
				utils.Warning("Failed to save hostname to config: %v", err)
			}
		}

		// Perform web login
		sessionToken, err := loginWithWeb(hostname)

		if err != nil {
			utils.Error("%v", err)
			return
		}

		if sessionToken == "" {
			utils.Error("Login appeared successful but no session token was received.")
			return
		}

		// Save session token to config
		if err := secrets.SaveSessionToken(sessionToken); err != nil {
			utils.Error("Failed to save session token: %v", err)
			return
		}

		// Update the global API client (always initialized)
		// Update base URL and token (hostname already includes protocol)
		apiBaseURL := hostname + "/api/v1"
		api.GlobalClient.SetBaseURL(apiBaseURL)
		api.GlobalClient.SetToken(sessionToken)

		utils.Success("Device authorized")
		fmt.Println()

		// Get user information
		var user *api.User
		user, err = api.GlobalClient.GetUser()
		if err != nil {
			utils.Warning("Failed to get user information: %v", err)
		} else {
			// Store userId and email in viper config
			viper.Set("userId", user.UserID)
			viper.Set("email", user.Email)
			if err := viper.WriteConfig(); err != nil {
				utils.Warning("Failed to save user information to config: %v", err)
			}

			// Ensure OLM credentials exist and are valid
			userID := user.UserID
			if err := utils.EnsureOlmCredentials(userID); err != nil {
				utils.Warning("Failed to ensure OLM credentials: %v", err)
			}
		}

		// List and select organization
		if user != nil {
			if _, err := utils.SelectOrg(user.UserID); err != nil {
				utils.Warning("%v", err)
			}
		}

		// Print logged in message after all setup is complete
		if user != nil {
			displayName := user.Email
			if displayName == "" && user.Username != nil && *user.Username != "" {
				displayName = *user.Username
			}
			if displayName != "" {
				utils.Success("Logged in as %s", displayName)
			}
		}
	},
}
