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

	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/secrets"
	"github.com/fosrl/cli/internal/tui"
	"github.com/fosrl/cli/internal/utils"
	versionpkg "github.com/fosrl/cli/internal/version"
	"github.com/fosrl/newt/logger"
	olmpkg "github.com/fosrl/olm/olm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultMTU           = 1280
	defaultDNS           = "8.8.8.8"
	defaultInterfaceName = "pangolin"
	defaultLogLevel      = "info"
	defaultEnableAPI     = true
	defaultSocketPath    = "/var/run/olm.sock"
	defaultPingInterval  = "5s"
	defaultPingTimeout   = "5s"
	defaultHolepunch     = true
	defaultVersion       = "Pangolin CLI"
	defaultOverrideDNS   = true
)

var (
	flagID            string
	flagSecret        string
	flagEndpoint      string
	flagOrgID         string
	flagMTU           int
	flagDNS           string
	flagInterfaceName string
	flagLogLevel      string
	flagHTTPAddr      string
	flagPingInterval  string
	flagPingTimeout   string
	flagHolepunch     bool
	flagTlsClientCert string
	flagAttached      bool
	flagSilent        bool
	flagOverrideDNS   bool
	flagUpstreamDNS   []string
)

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start a client connection",
	Long:  "Bring up a client tunneled connection",
	Run: func(cmd *cobra.Command, args []string) {

		if runtime.GOOS == "windows" {
			utils.Error("Windows is not supported")
			os.Exit(1)
		}

		// Check if a client is already running
		olmClient := olm.NewClient("")
		if olmClient.IsRunning() {
			utils.Info("A client is already running")
			os.Exit(1)
		}

		var olmID, olmSecret string
		var credentialsFromKeyring bool
		var userID string

		if flagID != "" && flagSecret != "" {
			// Use provided flags - no user session needed, continue even if not logged in
			// Org cannot be set when passing id and secret directly
			if flagOrgID != "" {
				utils.Error("--org cannot be set when passing --id and --secret directly")
				os.Exit(1)
			}
			olmID = flagID
			olmSecret = flagSecret
			credentialsFromKeyring = false
		} else if flagID != "" || flagSecret != "" {
			// If only one flag is provided, require both
			utils.Error("Both --id and --secret must be provided together")
			os.Exit(1)
		} else {
			// No flags provided - assume user is logged in and use credentials from config
			// Ensure user is logged in (this also verifies user exists via API)
			if err := utils.EnsureLoggedIn(); err != nil {
				utils.Error("%v", err)
				os.Exit(1)
			}

			// Get userId from viper (required for OLM credentials lookup)
			userID = viper.GetString("userId")
			if userID == "" {
				utils.Error("Please log in first. Run `pangolin login` to login")
				os.Exit(1)
			}

			// Ensure OLM credentials exist and are valid
			var err error
			if err = utils.EnsureOlmCredentials(userID); err != nil {
				utils.Error("Failed to ensure OLM credentials: %v", err)
				os.Exit(1)
			}

			// Get OLM credentials from config (they should exist after EnsureOlmCredentials)
			olmID, olmSecret, err = secrets.GetOlmCredentials(userID)
			if err != nil {
				utils.Error("Failed to get OLM credentials: %v", err)
				os.Exit(1)
			}
			credentialsFromKeyring = true
		}

		// Get orgId from flag or viper (required for OLM config when using logged-in user)
		var orgID string
		if credentialsFromKeyring {
			// When using credentials from keyring, orgID is required
			orgID = flagOrgID
			if orgID == "" {
				orgID = viper.GetString("orgId")
			}
			if orgID == "" {
				utils.Error("Please select an organization first. Run `pangolin select org` to select an organization or pass --org [id] to the command")
				os.Exit(1)
			}
		} else {
			// When using id/secret directly, orgID is optional (may come from credentials)
			orgID = flagOrgID
			if orgID == "" {
				orgID = viper.GetString("orgId")
			}
			// orgID is optional when using direct credentials
		}

		// Ensure org access (only when using logged-in user, not when credentials come from flags)
		if credentialsFromKeyring && userID != "" {
			if err := utils.EnsureOrgAccess(orgID, userID); err != nil {
				utils.Error("%v", err)
				os.Exit(1)
			}
		}

		// Handle log file setup - if detached mode, always use log file
		var logFile string
		if !flagAttached {
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

			// Add org flag (required for subprocess, which runs as root and won't have user's config)
			// Use flag value if provided, otherwise use the resolved orgID
			// Only add org flag if credentials came from keyring (not when id/secret are provided directly)
			if credentialsFromKeyring {
				if flagOrgID != "" {
					cmdArgs = append(cmdArgs, "--org", flagOrgID)
				} else {
					cmdArgs = append(cmdArgs, "--org", orgID)
				}
			}

			// Add all flags that were set (except --attach)
			// OLM credentials are always included (from flags, config, or newly created)
			cmdArgs = append(cmdArgs, "--id", olmID)
			cmdArgs = append(cmdArgs, "--secret", olmSecret)

			// Always pass endpoint to subprocess (required, subprocess won't have user's config)
			// Get endpoint from flag or hostname config (same logic as attached mode)
			endpoint := flagEndpoint
			if endpoint == "" {
				// Check if hostname is actually set in config (not just using default)
				if hostname := viper.GetString("hostname"); hostname != "" {
					endpoint = hostname
				}
			}
			if endpoint == "" {
				utils.Error("Endpoint is required. Please login with a host or provide --endpoint flag")
				os.Exit(1)
			}
			cmdArgs = append(cmdArgs, "--endpoint", endpoint)

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
			if cmd.Flags().Changed("http-addr") {
				cmdArgs = append(cmdArgs, "--http-addr", flagHTTPAddr)
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
				} else {
					cmdArgs = append(cmdArgs, "--holepunch=false")
				}
			}
			if cmd.Flags().Changed("tls-client-cert") {
				cmdArgs = append(cmdArgs, "--tls-client-cert", flagTlsClientCert)
			}
			if cmd.Flags().Changed("override-dns") {
				if flagOverrideDNS {
					cmdArgs = append(cmdArgs, "--override-dns")
				} else {
					cmdArgs = append(cmdArgs, "--override-dns=false")
				}
			}
			if cmd.Flags().Changed("upstream-dns") {
				// For string slice flags, we need to pass each value separately
				// Cobra's StringSliceVar supports multiple --upstream-dns flags or comma-separated values
				for _, dns := range flagUpstreamDNS {
					cmdArgs = append(cmdArgs, "--upstream-dns", dns)
				}
			}

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
				// Export environment variable to indicate credentials came from config
				// This allows subprocess to distinguish between user-provided credentials and stored credentials
				shellCmd := ""
				if credentialsFromKeyring {
					shellCmd = "export PANGOLIN_CREDENTIALS_FROM_KEYRING=1 && "
				}
				// Build command: nohup executable args >/dev/null 2>&1 &
				shellCmd += "nohup"
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

			// In silent mode, skip TUI and just exit after starting the process
			if flagSilent {
				os.Exit(0)
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
						return "Starting"
					} else if status.Registered {
						return "Registered"
					}
					return "Starting"
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

		// Get endpoint from flag or config - required
		endpoint := flagEndpoint
		if endpoint == "" {
			// Check if hostname is actually set in config (not just using default)
			if hostname := viper.GetString("hostname"); hostname != "" {
				endpoint = hostname
			}
		}
		if endpoint == "" {
			utils.Error("Endpoint is required. Please provide --endpoint flag or set hostname in config")
			os.Exit(1)
		}

		mtu := getInt(flagMTU, "mtu", "mtu", defaultMTU)
		dns := getString(flagDNS, "dns", "dns", defaultDNS)
		interfaceName := getString(flagInterfaceName, "interface-name", "interface_name", defaultInterfaceName)
		logLevel := getString(flagLogLevel, "log-level", "log_level", defaultLogLevel)
		enableAPI := defaultEnableAPI

		// In detached mode, API cannot be disabled (required for status/control)
		if !flagAttached && !enableAPI {
			enableAPI = true
		}

		httpAddr := getString(flagHTTPAddr, "http-addr", "http_addr", "")
		socketPath := defaultSocketPath
		pingInterval := getString(flagPingInterval, "ping-interval", "ping_interval", defaultPingInterval)
		pingTimeout := getString(flagPingTimeout, "ping-timeout", "ping_timeout", defaultPingTimeout)
		holepunch := getBool(flagHolepunch, "holepunch", "holepunch", defaultHolepunch)
		tlsClientCert := getString(flagTlsClientCert, "tls-client-cert", "tls_client_cert", "")
		version := versionpkg.Version
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

		// Get UserToken from config if credentials came from config
		// Check environment variable to distinguish between:
		// - Parent process passing id/secret from config (should fetch userToken)
		// - User directly passing id/secret (should NOT fetch userToken)
		var userToken string
		credentialsFromKeyringEnv := os.Getenv("PANGOLIN_CREDENTIALS_FROM_KEYRING")
		if credentialsFromKeyringEnv == "1" || credentialsFromKeyring {
			// Credentials came from config, fetch userToken from secrets
			token, err := secrets.GetSessionToken()
			if err != nil {
				utils.Warning("Failed to get session token: %v", err)
			} else {
				userToken = token
			}
		}

		// Create context for signal handling and cleanup
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer olmpkg.Close()
		defer stop()

		// Create OLM GlobalConfig with hardcoded values from Swift
		olmInitConfig := olmpkg.GlobalConfig{
			LogLevel:   logLevel,
			EnableAPI:  enableAPI,
			SocketPath: socketPath,
			HTTPAddr:   httpAddr,
			Version:    version,
			Agent:      defaultVersion,
			OnTerminated: func() {
				utils.Info("Client process terminated")
				stop()
				os.Exit(0)
			},
			OnAuthError: func(statusCode int, message string) {
				utils.Error("Authentication error: %d %s", statusCode, message)
				stop()
				os.Exit(1)
			},
			OnExit: func() {
				utils.Info("Client process exiting")
				os.Exit(0)
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

		// Add UserToken if we have it (from flag or config)
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

		olmpkg.Init(ctx, olmInitConfig)
		if enableAPI {
			olmpkg.StartApi()
		}
		olmpkg.StartTunnel(olmConfig)
	},
}

// addClientFlags adds all client flags to the given command
func addClientFlags(cmd *cobra.Command) {
	// Optional flags - if not provided, will use config or create new OLM
	cmd.Flags().StringVar(&flagID, "id", "", "Client ID (optional, will use user info if not provided)")
	cmd.Flags().StringVar(&flagSecret, "secret", "", "Client secret (optional, will use user info if not provided)")

	// Optional flags
	cmd.Flags().StringVar(&flagOrgID, "org", "", "Organization ID (optional, will use selected org if not provided)")
	cmd.Flags().StringVar(&flagEndpoint, "endpoint", "", "Client endpoint (required if not logged in)")
	cmd.Flags().IntVar(&flagMTU, "mtu", 0, fmt.Sprintf("MTU (default: %d)", defaultMTU))
	cmd.Flags().StringVar(&flagDNS, "dns", "", fmt.Sprintf("DNS server (default: %s)", defaultDNS))
	cmd.Flags().StringVar(&flagInterfaceName, "interface-name", "", fmt.Sprintf("Interface name (default: %s)", defaultInterfaceName))
	cmd.Flags().StringVar(&flagLogLevel, "log-level", "", fmt.Sprintf("Log level (default: %s)", defaultLogLevel))
	cmd.Flags().StringVar(&flagHTTPAddr, "http-addr", "", "HTTP address")
	cmd.Flags().StringVar(&flagPingInterval, "ping-interval", "", fmt.Sprintf("Ping interval (default: %s)", defaultPingInterval))
	cmd.Flags().StringVar(&flagPingTimeout, "ping-timeout", "", fmt.Sprintf("Ping timeout (default: %s)", defaultPingTimeout))
	cmd.Flags().BoolVar(&flagHolepunch, "holepunch", false, fmt.Sprintf("Enable holepunching (default: %v)", defaultHolepunch))
	cmd.Flags().StringVar(&flagTlsClientCert, "tls-client-cert", "", "TLS client certificate path")
	cmd.Flags().BoolVar(&flagOverrideDNS, "override-dns", defaultOverrideDNS, fmt.Sprintf("Override system DNS for resolving internal resource alias (default: %v)", defaultOverrideDNS))
	cmd.Flags().StringSliceVar(&flagUpstreamDNS, "upstream-dns", nil, fmt.Sprintf("List of DNS servers to use for external DNS resolution if overriding system DNS (default: %s)", defaultDNS))
	cmd.Flags().BoolVar(&flagAttached, "attach", false, "Run in attached mode (foreground, default is detached)")
	cmd.Flags().BoolVar(&flagSilent, "silent", false, "Disable TUI and run silently (only applies to detached mode)")
}

func init() {
	addClientFlags(ClientCmd)
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
