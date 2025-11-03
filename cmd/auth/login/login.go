package login

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type HostingOption string

const (
	HostingOptionCloud      HostingOption = "cloud"
	HostingOptionSelfHosted HostingOption = "self-hosted"
)

type LoginMethod string

const (
	LoginMethodCredentials LoginMethod = "credentials"
	LoginMethodWeb         LoginMethod = "web"
)

func loginWithCredentials(hostname string) (string, error) {
	// Build base URL for login (use hostname as-is, LoginWithCookie will add /api/v1/auth/login)
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

	// Prompt for email and password
	var email, password string
	credentialsForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Email").
				Placeholder("your.email@example.com").
				Value(&email),
			huh.NewInput().
				Title("Password").
				Placeholder("Enter your password").
				EchoMode(huh.EchoModePassword).
				Value(&password),
		),
	)

	if err := credentialsForm.Run(); err != nil {
		return "", fmt.Errorf("error collecting credentials: %w", err)
	}

	// Perform login
	loginReq := api.LoginRequest{
		Email:    email,
		Password: password,
	}

	loginResp, sessionToken, err := api.LoginWithCookie(loginClient, loginReq)
	if err != nil {
		return "", err
	}

	// Handle nil response (shouldn't happen, but be safe)
	if loginResp == nil {
		if sessionToken != "" {
			// If we got a token, consider it successful
			return sessionToken, nil
		}
		return "", fmt.Errorf("login failed - no response received")
	}

	// Handle different response scenarios
	if loginResp.TwoFactorSetupRequired {
		return "", fmt.Errorf("two-factor authentication setup is required. Please complete setup in the web interface")
	}

	if loginResp.UseSecurityKey {
		return "", fmt.Errorf("security key authentication is required. This is not yet supported in the CLI")
	}

	if loginResp.CodeRequested {
		// Prompt for 2FA code
		var code string
		codeForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Two-factor authentication code").
					Placeholder("Enter your 2FA code").
					Value(&code),
			),
		)

		if err := codeForm.Run(); err != nil {
			return "", fmt.Errorf("error collecting 2FA code: %w", err)
		}

		// Retry login with code
		loginReq.Code = code
		loginResp, sessionToken, err = api.LoginWithCookie(loginClient, loginReq)
		if err != nil {
			return "", err
		}
	}

	if loginResp.EmailVerificationRequired {
		utils.Info("Email verification is required. Please check your email and verify your account.")
		// Still save the token if we got one
		if sessionToken != "" {
			return sessionToken, nil
		}
		return "", fmt.Errorf("email verification required but no session token received")
	}

	return sessionToken, nil
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Run()
}

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
	expiresAt := startResp.ExpiresAt

	// Build the login URL
	loginURL := strings.TrimSuffix(hostname, "/") + "/auth/login/device"

	// Display code and instructions (similar to GH CLI format)
	utils.Info("First copy your one-time code: %s", code)
	utils.Info("Press Enter to open %s in your browser...", loginURL)

	// Wait for user to press Enter
	reader := bufio.NewReader(os.Stdin)
	_, err = reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	// Open browser
	if err := openBrowser(loginURL); err != nil {
		// Don't fail if browser can't be opened, just warn
		utils.Warning("Failed to open browser automatically: %v", err)
		utils.Info("Please manually visit: %s", loginURL)
	}

	// Poll for verification
	pollInterval := 3 * time.Second // Poll every 2 seconds
	startTime := time.Now()
	maxPollDuration := 5 * time.Minute // Maximum polling duration (5 minutes)

	var token string

	for {
		// Check if code has expired
		now := time.Now().UnixMilli()
		if now >= expiresAt {
			return "", fmt.Errorf("code expired. Please try again")
		}

		// Check if we've exceeded max polling duration
		if time.Since(startTime) > maxPollDuration {
			return "", fmt.Errorf("polling timeout. Please try again")
		}

		// Poll for verification status
		pollResp, message, err := api.PollDeviceWebAuth(loginClient, code)
		if err != nil {
			// Check if it's a rate limit error (429)
			if errorResp, ok := err.(*api.ErrorResponse); ok && errorResp.Status == 429 {
				// Rate limited - wait a bit longer before retrying
				utils.Debug("Rate limited, waiting before retry...")
				time.Sleep(10 * time.Second)
				continue
			}

			// Check if it's an IP mismatch error (403)
			if errorResp, ok := err.(*api.ErrorResponse); ok && errorResp.Status == 403 {
				return "", fmt.Errorf("IP address mismatch. Your IP address may have changed. Please try again")
			}

			// For other errors, return them
			return "", fmt.Errorf("failed to poll device web auth: %w", err)
		}

		// Check verification status
		if pollResp.Verified {
			token = pollResp.Token
			if token == "" {
				return "", fmt.Errorf("verification succeeded but no token received")
			}
			return token, nil
		}

		// Check for expired or not found messages
		if message == "Code expired" || message == "Code not found" {
			return "", fmt.Errorf("code expired or not found. Please try again")
		}

		// Wait before next poll
		time.Sleep(pollInterval)
	}
}

var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Pangolin",
	Long:  "Interactive login to select your hosting option and configure access.",
	Run: func(cmd *cobra.Command, args []string) {
		var hostingOption HostingOption
		var hostname string
		var loginMethod LoginMethod

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
			homeDir, err := os.UserHomeDir()
			if err == nil {
				viper.SetConfigFile(homeDir + "/.pangolin.yaml")
				viper.SetConfigType("yaml")
			}
		}

		if err := viper.WriteConfig(); err != nil {
			// If config file doesn't exist, create it
			if err := viper.SafeWriteConfig(); err != nil {
				utils.Warning("Failed to save hostname to config: %v", err)
			}
		}

		// Select login method
		loginMethodForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[LoginMethod]().
					Title("Select login method").
					Options(
						huh.NewOption("Login with web (recommended)", LoginMethodWeb),
						huh.NewOption("Login with credentials", LoginMethodCredentials),
					).
					Value(&loginMethod),
			),
		)

		if err := loginMethodForm.Run(); err != nil {
			utils.Error("Error: %v", err)
			return
		}

		// Branch based on login method
		var sessionToken string
		var err error

		if loginMethod == LoginMethodWeb {
			sessionToken, err = loginWithWeb(hostname)
		} else {
			sessionToken, err = loginWithCredentials(hostname)
		}

		if err != nil {
			utils.Error("%v", err)
			return
		}

		if sessionToken == "" {
			utils.Error("Login appeared successful but no session token was received.")
			return
		}

		// Save session token to keyring
		if err := api.SaveSessionToken(sessionToken); err != nil {
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
			if displayName == "" && user.Username != "" {
				displayName = user.Username
			}
			if displayName != "" {
				utils.Success("Logged in as %s", displayName)
			}
		}
	},
}
