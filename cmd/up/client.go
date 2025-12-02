package up

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/tui"
	"github.com/fosrl/cli/internal/utils"
	"github.com/fosrl/newt/logger"
	olmpkg "github.com/fosrl/olm/olm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultMTU           = 1280
	defaultDNS           = "8.8.8.8"
	defaultInterfaceName = "Pangolin"
	defaultLogLevel      = "info"
	defaultEnableAPI     = true
	defaultSocketPath    = "/var/run/olm.sock"
	defaultPingInterval  = "5s"
	defaultPingTimeout   = "5s"
	defaultHolepunch     = false
	defaultVersion       = "Pangolin CLI"
	defaultOverrideDNS   = true
)

var (
	flagID            string
	flagSecret        string
	flagEndpoint      string
	flagMTU           int
	flagDNS           string
	flagInterfaceName string
	flagLogLevel      string
	flagEnableAPI     bool
	flagHTTPAddr      string
	flagSocketPath    string
	flagPingInterval  string
	flagPingTimeout   string
	flagHolepunch     bool
	flagTlsClientCert string
	flagVersion       string
	flagAttached      bool
	flagLogFile       string
	flagUserToken     string
	flagSkipUserToken bool
	flagOverrideDNS   bool
	flagUpstreamDNS   []string
)

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start a client connection",
	Long:  "Bring up a client tunneled connection",
	Run: func(cmd *cobra.Command, args []string) {

		if runtime.GOOS == "windows" {
			utils.Error("Windows is not supported for detached mode")
			os.Exit(1)
		}

		// Check if a client is already running
		olmClient := olm.NewClient("")
		if olmClient.IsRunning() {
			// Try to get status to confirm it's actually running
			status, err := olmClient.GetStatus()
			if err == nil && status != nil {
				utils.Info("A client is already running")
				os.Exit(1)
			}
			// If status check fails but socket exists, still warn
			utils.Error("A client appears to be running (socket exists)")
			os.Exit(1)
		}

		// Get OLM credentials: from flags, or keyring, or create new
		var olmID, olmSecret string
		var credentialsFromKeyring bool
		if flagID != "" && flagSecret != "" {
			// Use provided flags
			olmID = flagID
			olmSecret = flagSecret
			credentialsFromKeyring = false
		} else if flagID != "" || flagSecret != "" {
			// If only one flag is provided, require both
			utils.Error("Both --id and --secret must be provided together, or neither (to use keyring or create new)")
			os.Exit(1)
		} else {
			// Ensure user is logged in before getting/creating OLM credentials
			if err := utils.EnsureLoggedIn(); err != nil {
				utils.Error("%v", err)
				os.Exit(1)
			}

			// Get userId from viper (required for OLM credentials keyring lookup)
			userID := viper.GetString("userId")
			if userID == "" {
				utils.Error("UserId is required. Please log in first")
				os.Exit(1)
			}

			// Try to get from keyring
			var err error
			olmID, olmSecret, err = api.GetOlmCredentials(userID)
			if err != nil {
				// Not found in keyring, create new OLM
				deviceName := getDeviceName()
				defaultOlmName := fmt.Sprintf("%s", deviceName)

				// Prompt user to edit the name with pre-filled default
				olmName := defaultOlmName
				nameForm := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Client name").
							Description("Enter a name for this client (press Enter to use default)").
							Value(&olmName),
					),
				)

				if err := nameForm.Run(); err != nil {
					utils.Error("Error: failed to collect client name: %v", err)
					os.Exit(1)
				}

				// Use default if user cleared the name
				if strings.TrimSpace(olmName) == "" {
					olmName = defaultOlmName
				} else {
					olmName = strings.TrimSpace(olmName)
				}

				response, err := api.GlobalClient.CreateOlm(olmName, userID)
				if err != nil {
					utils.Error("Failed to create OLM: %v", err)
					os.Exit(1)
				}

				// Save to keyring
				if err := api.SaveOlmCredentials(userID, response.OlmID, response.Secret); err != nil {
					utils.Warning("Failed to save OLM credentials to keyring: %v", err)
				}

				olmID = response.OlmID
				olmSecret = response.Secret
				credentialsFromKeyring = false
			} else {
				// Successfully retrieved from keyring
				credentialsFromKeyring = true
			}
		}

		// Get orgId from viper (required for OLM config)
		orgID := viper.GetString("orgId")
		if orgID == "" {
			utils.Error("OrgId is required. Please select an organization first")
			os.Exit(1)
		}

		// Get UserToken if credentials came from keyring (must happen in parent process)
		// This cannot happen in the subprocess because it runs as root and can't access user's keyring
		var userToken string
		if credentialsFromKeyring && !flagSkipUserToken {
			token, err := api.GetSessionToken()
			if err != nil {
				utils.Warning("Failed to get session token: %v", err)
			} else {
				userToken = token
			}
		}

		// Handle log file setup - if detached mode, always use log file
		logFile := flagLogFile
		if !flagAttached && logFile == "" {
			logFile = utils.GetDefaultLogPath()
		}

		// Handle detached mode - subprocess self without --attach flag
		// Skip detached mode if already running as root (we're a subprocess spawned by sudo)
		isRunningAsRoot := runtime.GOOS != "windows" && os.Geteuid() == 0
		if !flagAttached && !isRunningAsRoot {
			executable, err := os.Executable()
			if err != nil {
				utils.Error("Error: failed to get executable path: %v", err)
				os.Exit(1)
			}

			// Build command arguments, excluding --attach flag
			cmdArgs := []string{"up", "client"}

			// Add orgId flag (required for subprocess, which runs as root and won't have user's config)
			cmdArgs = append(cmdArgs, "--orgId", orgID)

			// Add all flags that were set (except --attach)
			// OLM credentials are always included (from flags, keyring, or newly created)
			cmdArgs = append(cmdArgs, "--id", olmID)
			cmdArgs = append(cmdArgs, "--secret", olmSecret)

			// Always pass endpoint to subprocess (required, subprocess won't have user's config)
			// Get endpoint from flag or hostname config (same logic as attached mode)
			endpoint := flagEndpoint
			if endpoint == "" {
				endpoint = viper.GetString("hostname")
			}
			if endpoint != "" {
				cmdArgs = append(cmdArgs, "--endpoint", endpoint)
			}

			// Pass UserToken to subprocess if we have it (from keyring in parent process)
			if userToken != "" {
				cmdArgs = append(cmdArgs, "--user-token", userToken)
			}
			if cmd.Flags().Changed("skip-user-token") {
				if flagSkipUserToken {
					cmdArgs = append(cmdArgs, "--skip-user-token")
				}
			}

			// Optional flags - only include if they were explicitly set
			if cmd.Flags().Changed("mtu") {
				cmdArgs = append(cmdArgs, "--mtu", fmt.Sprintf("%d", flagMTU))
			}
			if cmd.Flags().Changed("dns") {
				cmdArgs = append(cmdArgs, "--dns", flagDNS)
			}
			if cmd.Flags().Changed("interface-name") {
				cmdArgs = append(cmdArgs, "--interface-name", flagInterfaceName)
			}
			if cmd.Flags().Changed("log-level") {
				cmdArgs = append(cmdArgs, "--log-level", flagLogLevel)
			}
			if cmd.Flags().Changed("enable-api") {
				if flagEnableAPI {
					cmdArgs = append(cmdArgs, "--enable-api")
				}
			}
			if cmd.Flags().Changed("http-addr") {
				cmdArgs = append(cmdArgs, "--http-addr", flagHTTPAddr)
			}
			if cmd.Flags().Changed("socket-path") {
				cmdArgs = append(cmdArgs, "--socket-path", flagSocketPath)
			}
			if cmd.Flags().Changed("ping-interval") {
				cmdArgs = append(cmdArgs, "--ping-interval", flagPingInterval)
			}
			if cmd.Flags().Changed("ping-timeout") {
				cmdArgs = append(cmdArgs, "--ping-timeout", flagPingTimeout)
			}
			if cmd.Flags().Changed("holepunch") {
				if flagHolepunch {
					cmdArgs = append(cmdArgs, "--holepunch")
				}
			}
			if cmd.Flags().Changed("tls-client-cert") {
				cmdArgs = append(cmdArgs, "--tls-client-cert", flagTlsClientCert)
			}
			if cmd.Flags().Changed("version") {
				cmdArgs = append(cmdArgs, "--version", flagVersion)
			}
			if cmd.Flags().Changed("override-dns") {
				if flagOverrideDNS {
					cmdArgs = append(cmdArgs, "--override-dns")
				}
			}
			if cmd.Flags().Changed("upstream-dns") {
				// For string slice flags, we need to pass each value separately
				// Cobra's StringSliceVar supports multiple --upstream-dns flags or comma-separated values
				for _, dns := range flagUpstreamDNS {
					cmdArgs = append(cmdArgs, "--upstream-dns", dns)
				}
			}
			// Always add log-file when detached (use default if not explicitly set)
			cmdArgs = append(cmdArgs, "--log-file", logFile)

			// Add positional args if any
			cmdArgs = append(cmdArgs, args...)

			// Create command - subprocess should run with elevated permissions
			var procCmd *exec.Cmd
			if runtime.GOOS != "windows" {
				// Use sudo with a shell wrapper to background the subprocess
				// This allows sudo to exit immediately after starting the subprocess
				// The subprocess needs root access for network interface creation
				// Build shell command with proper quoting using printf %q
				var shellArgs []string
				shellArgs = append(shellArgs, executable)
				shellArgs = append(shellArgs, cmdArgs...)
				// Build command: nohup executable args >/dev/null 2>&1 &
				shellCmd := "nohup"
				for _, arg := range shellArgs {
					shellCmd += " " + fmt.Sprintf("%q", arg)
				}
				shellCmd += " >/dev/null 2>&1 &"
				procCmd = exec.Command("sudo", "sh", "-c", shellCmd)
				// Connect stdin/stderr so sudo can prompt for password interactively
				procCmd.Stdin = os.Stdin
				procCmd.Stdout = nil
				procCmd.Stderr = os.Stderr
			} else {
				utils.Error("Windows is not supported for detached mode")
				os.Exit(1)
			}

			// Start the process
			if err := procCmd.Start(); err != nil {
				utils.Error("Error: failed to start detached process: %v", err)
				os.Exit(1)
			}

			// Wait for sudo to complete (password prompt + subprocess start)
			// The shell wrapper backgrounds the subprocess, so sudo exits immediately
			if err := procCmd.Wait(); err != nil {
				utils.Error("Error: failed to start subprocess: %v", err)
				os.Exit(1)
			}

			// Show live log preview and status
			completed, err := tui.NewLogPreview(tui.LogPreviewConfig{
				LogFile: logFile,
				Header:  "Starting up client...",
				ExitCondition: func(client *olm.Client, status *olm.StatusResponse) (bool, bool) {
					// Exit when interface is registered
					if status != nil && status.Registered {
						return true, true
					}
					return false, false
				},
				OnEarlyExit: func(client *olm.Client) {
					// Kill the subprocess if user exits early
					if client.IsRunning() {
						client.Exit()
					}
				},
				StatusFormatter: func(isRunning bool, status *olm.StatusResponse) string {
					if !isRunning || status == nil {
						return fmt.Sprintf("Starting")
					} else if status.Registered {
						return fmt.Sprintf("Registered")
					}
					return fmt.Sprintf("Starting")
				},
			})
			if err != nil {
				utils.Error("Error: %v", err)
				os.Exit(1)
			}

			// Check if the process completed successfully or was killed
			if !completed {
				// User exited early - subprocess was killed
				utils.Info("Client process killed")
			} else {
				// Completed successfully
				utils.Success("Client interface created successfully")
			}
			os.Exit(0)
		}

		// Helper function to get value with precedence: CLI flag > default
		getString := func(flagValue, flagName, configKey, defaultValue string) string {
			// Check if flag was explicitly set (CLI takes precedence)
			if cmd.Flags().Changed(flagName) {
				return flagValue
			}
			return defaultValue
		}

		getInt := func(flagValue int, flagName, configKey string, defaultValue int) int {
			// Check if flag was explicitly set (CLI takes precedence)
			if cmd.Flags().Changed(flagName) {
				return flagValue
			}
			return defaultValue
		}

		getBool := func(flagValue bool, flagName, configKey string, defaultValue bool) bool {
			// Check if flag was explicitly set (CLI takes precedence)
			if cmd.Flags().Changed(flagName) {
				return flagValue
			}
			return defaultValue
		}

		getStringSlice := func(flagValue []string, flagName, configKey string, defaultValue []string) []string {
			// Check if flag was explicitly set (CLI takes precedence)
			if cmd.Flags().Changed(flagName) {
				return flagValue
			}
			return defaultValue
		}

		// Parse duration strings to time.Duration
		parseDuration := func(durationStr string, defaultDuration time.Duration) time.Duration {
			if durationStr == "" {
				return defaultDuration
			}
			d, err := time.ParseDuration(durationStr)
			if err != nil {
				utils.Warning("Invalid duration format '%s', using default: %v", durationStr, defaultDuration)
				return defaultDuration
			}
			return d
		}

		// Get endpoint from hostname
		endpoint := flagEndpoint
		if endpoint == "" {
			endpoint = viper.GetString("hostname")
		}

		// Get values with precedence: CLI flag > config > default
		mtu := getInt(flagMTU, "mtu", "mtu", defaultMTU)
		dns := getString(flagDNS, "dns", "dns", defaultDNS)
		interfaceName := getString(flagInterfaceName, "interface-name", "interface_name", defaultInterfaceName)
		logLevel := getString(flagLogLevel, "log-level", "log_level", defaultLogLevel)
		enableAPI := getBool(flagEnableAPI, "enable-api", "enable_api", defaultEnableAPI)
		httpAddr := getString(flagHTTPAddr, "http-addr", "http_addr", "")
		socketPath := getString(flagSocketPath, "socket-path", "socket_path", defaultSocketPath)
		pingInterval := getString(flagPingInterval, "ping-interval", "ping_interval", defaultPingInterval)
		pingTimeout := getString(flagPingTimeout, "ping-timeout", "ping_timeout", defaultPingTimeout)
		holepunch := getBool(flagHolepunch, "holepunch", "holepunch", defaultHolepunch)
		tlsClientCert := getString(flagTlsClientCert, "tls-client-cert", "tls_client_cert", "")
		version := getString(flagVersion, "version", "version", defaultVersion)
		overrideDNS := getBool(flagOverrideDNS, "override-dns", "override_dns", defaultOverrideDNS)
		upstreamDNS := getStringSlice(flagUpstreamDNS, "upstream-dns", "upstream_dns", []string{defaultDNS})

		// Process UpstreamDNS: append :53 to each DNS server if not already present
		processedUpstreamDNS := make([]string, 0, len(upstreamDNS))
		for _, dns := range upstreamDNS {
			dns = strings.TrimSpace(dns)
			if dns == "" {
				continue
			}
			// Append :53 if not already present
			if !strings.Contains(dns, ":") {
				dns = dns + ":53"
			}
			processedUpstreamDNS = append(processedUpstreamDNS, dns)
		}
		// If no DNS servers were provided, use default
		if len(processedUpstreamDNS) == 0 {
			processedUpstreamDNS = []string{defaultDNS + ":53"}
		}

		// Parse durations
		defaultPingIntervalDuration, _ := time.ParseDuration(defaultPingInterval)
		defaultPingTimeoutDuration, _ := time.ParseDuration(defaultPingTimeout)
		pingIntervalDuration := parseDuration(pingInterval, defaultPingIntervalDuration)
		pingTimeoutDuration := parseDuration(pingTimeout, defaultPingTimeoutDuration)

		// Setup log file if specified
		if logFile != "" {
			if err := setupLogFile(logFile); err != nil {
				utils.Error("Error: failed to setup log file: %v", err)
				os.Exit(1)
			}
		}

		// Get UserToken from flag (passed from parent process) or from keyring (if in attached mode)
		// Note: userToken is already declared in outer scope, but in attached mode we may need to fetch it
		if flagUserToken != "" {
			// UserToken was passed from parent process (detached mode)
			userToken = flagUserToken
		} else if credentialsFromKeyring && !flagSkipUserToken {
			// In attached mode, fetch from keyring (if not already set in parent process)
			if userToken == "" {
				token, err := api.GetSessionToken()
				if err != nil {
					utils.Warning("Failed to get session token: %v", err)
				} else {
					userToken = token
				}
			}
		}

		// Create OLM GlobalConfig with hardcoded values from Swift
		olmInitConfig := olmpkg.GlobalConfig{
			LogLevel:   logLevel,
			EnableAPI:  true,
			SocketPath: socketPath,
			HTTPAddr:   httpAddr,
			Version:    version,
			OnTerminated: func() {
				utils.Info("Client process terminated")
				os.Exit(0)
			},
			OnAuthError: func(statusCode int, message string) {
				utils.Error("Authentication error: %d %s", statusCode, message)
				os.Exit(1)
			},
		}

		olmConfig := olmpkg.TunnelConfig{
			Endpoint:             endpoint,
			ID:                   olmID,
			Secret:               olmSecret,
			OrgID:                orgID,
			MTU:                  mtu,
			DNS:                  dns,
			InterfaceName:        interfaceName,
			Holepunch:            holepunch,
			TlsClientCert:        tlsClientCert,
			PingIntervalDuration: pingIntervalDuration,
			PingTimeoutDuration:  pingTimeoutDuration,
			OverrideDNS:          overrideDNS,
			UpstreamDNS:          processedUpstreamDNS,
		}

		// Add UserToken if we have it (from flag or keyring)
		if userToken != "" {
			olmConfig.UserToken = userToken
		}

		// Check if running with elevated permissions (required for network interface creation)
		// This check is only for attached mode; in detached mode, the subprocess runs elevated
		if runtime.GOOS != "windows" {
			if os.Geteuid() != 0 {
				utils.Error("This command requires elevated permissions for network interface creation.")
				utils.Info("Please run with sudo or use detached mode (default) to run the subprocess elevated.")
				os.Exit(1)
			}
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		olmpkg.Init(ctx, olmInitConfig)
		if enableAPI {
			olmpkg.StartApi()
		}
		olmpkg.StartTunnel(olmConfig)
	},
}

// getDeviceName returns a human-readable device name
func getDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "Unknown Device"
	}
	return hostname
}

func init() {
	// Optional flags - if not provided, will use keyring or create new OLM
	ClientCmd.Flags().StringVar(&flagID, "id", "", "Client ID (optional, will use keyring or create new if not provided)")
	ClientCmd.Flags().StringVar(&flagSecret, "secret", "", "Client secret (optional, will use keyring or create new if not provided)")

	// Optional flags
	ClientCmd.Flags().StringVar(&flagEndpoint, "endpoint", "", "Client endpoint (defaults to hostname from config)")
	ClientCmd.Flags().IntVar(&flagMTU, "mtu", 0, fmt.Sprintf("MTU (default: %d)", defaultMTU))
	ClientCmd.Flags().StringVar(&flagDNS, "dns", "", fmt.Sprintf("DNS server (default: %s)", defaultDNS))
	ClientCmd.Flags().StringVar(&flagInterfaceName, "interface-name", "", fmt.Sprintf("Interface name (default: %s)", defaultInterfaceName))
	ClientCmd.Flags().StringVar(&flagLogLevel, "log-level", "", fmt.Sprintf("Log level (default: %s)", defaultLogLevel))
	ClientCmd.Flags().BoolVar(&flagEnableAPI, "enable-api", false, fmt.Sprintf("Enable API (default: %v)", defaultEnableAPI))
	ClientCmd.Flags().StringVar(&flagHTTPAddr, "http-addr", "", "HTTP address")
	ClientCmd.Flags().StringVar(&flagSocketPath, "socket-path", "", fmt.Sprintf("Socket path (default: %s)", defaultSocketPath))
	ClientCmd.Flags().StringVar(&flagPingInterval, "ping-interval", "", fmt.Sprintf("Ping interval (default: %s)", defaultPingInterval))
	ClientCmd.Flags().StringVar(&flagPingTimeout, "ping-timeout", "", fmt.Sprintf("Ping timeout (default: %s)", defaultPingTimeout))
	ClientCmd.Flags().BoolVar(&flagHolepunch, "holepunch", false, fmt.Sprintf("Enable holepunching (default: %v)", defaultHolepunch))
	ClientCmd.Flags().StringVar(&flagTlsClientCert, "tls-client-cert", "", "TLS client certificate path")
	ClientCmd.Flags().StringVar(&flagVersion, "version", "", fmt.Sprintf("Version (default: %s)", defaultVersion))
	ClientCmd.Flags().BoolVar(&flagOverrideDNS, "override-dns", defaultOverrideDNS, fmt.Sprintf("Override system DNS for resolving internal resource alias (default: %v)", defaultOverrideDNS))
	ClientCmd.Flags().StringSliceVar(&flagUpstreamDNS, "upstream-dns", nil, fmt.Sprintf("List of DNS servers to use for external DNS resolution if overriding system DNS (default: %s)", defaultDNS))
	ClientCmd.Flags().BoolVar(&flagAttached, "attach", false, "Run in attached mode (foreground, default is detached)")
	ClientCmd.Flags().StringVar(&flagLogFile, "log-file", "", "Path to log file (defaults to standard log location when detached)")
	ClientCmd.Flags().StringVar(&flagUserToken, "user-token", "", "User session token (if not provided, will be retrieved from keyring)")
	ClientCmd.Flags().BoolVar(&flagSkipUserToken, "skip-user-token", false, "Skip user token retrieval from keyring (run without user token)")

	UpCmd.AddCommand(ClientCmd)
}

// setupLogFile sets up file logging with rotation
func setupLogFile(logPath string) error {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Rotate log file if needed
	err = rotateLogFile(logDir, logPath)
	if err != nil {
		// Log warning but continue
		log.Printf("Warning: failed to rotate log file: %v", err)
	}

	// Open log file for appending
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// Set the logger output
	logger.GetLogger().SetOutput(file)

	log.Printf("Logging to file: %s", logPath)
	return nil
}

// rotateLogFile handles daily log rotation
func rotateLogFile(logDir string, logFile string) error {
	// Get current log file info
	info, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No current log file to rotate
		}
		return fmt.Errorf("failed to stat log file: %v", err)
	}

	// Check if log file is from today
	now := time.Now()
	fileTime := info.ModTime()

	// If the log file is from today, no rotation needed
	if now.Year() == fileTime.Year() && now.YearDay() == fileTime.YearDay() {
		return nil
	}

	// Create rotated filename with date
	rotatedName := fmt.Sprintf("client-%s.log", fileTime.Format("2006-01-02"))
	rotatedPath := filepath.Join(logDir, rotatedName)

	// Rename current log file to dated filename
	err = os.Rename(logFile, rotatedPath)
	if err != nil {
		return fmt.Errorf("failed to rotate log file: %v", err)
	}

	// Clean up old log files (keep last 30 days)
	cleanupOldLogFiles(logDir, 30)
	return nil
}

// cleanupOldLogFiles removes log files older than specified days
func cleanupOldLogFiles(logDir string, daysToKeep int) {
	cutoff := time.Now().AddDate(0, 0, -daysToKeep)
	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "client-") && strings.HasSuffix(file.Name(), ".log") {
			filePath := filepath.Join(logDir, file.Name())
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(filePath)
			}
		}
	}
}
