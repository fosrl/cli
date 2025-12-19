package client

import (
	"context"
	"errors"
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

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/tui"
	"github.com/fosrl/cli/internal/utils"
	versionpkg "github.com/fosrl/cli/internal/version"
	newtLogger "github.com/fosrl/newt/logger"
	olmpkg "github.com/fosrl/olm/olm"
	"github.com/spf13/cobra"
)

const (
	defaultDNSServer  = "8.8.8.8"
	defaultEnableAPI  = true
	defaultSocketPath = "/var/run/olm.sock"
	defaultAgent      = "Pangolin CLI"
)

type ClientUpCmdOpts struct {
	ID            string
	Secret        string
	Endpoint      string
	OrgID         string
	MTU           int
	DNS           string
	InterfaceName string
	LogLevel      string
	HTTPAddr      string
	PingInterval  time.Duration
	PingTimeout   time.Duration
	Holepunch     bool
	TlsClientCert string
	Attached      bool
	Silent        bool
	OverrideDNS   bool
	TunnelDNS     bool
	UpstreamDNS   []string
}

func ClientUpCmd() *cobra.Command {
	opts := ClientUpCmdOpts{}

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Start a client connection",
		Long:  "Bring up a client tunneled connection",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// `--id` and `--secret` must be specified together
			if (opts.ID == "") != (opts.Secret == "") {
				return errors.New("--id and --secret must be provided together")
			}

			if opts.Attached && opts.Silent {
				return errors.New("--silent and --attached options conflict")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := clientUpMain(cmd, &opts, args); err != nil {
				os.Exit(1)
			}
		},
	}

	// Optional flags - if not provided, will use config or create new OLM
	cmd.Flags().StringVar(&opts.ID, "id", "", "Client ID (optional, will use user info if not provided)")
	cmd.Flags().StringVar(&opts.Secret, "secret", "", "Client secret (optional, will use user info if not provided)")

	// Optional flags
	cmd.Flags().StringVar(&opts.OrgID, "org", "", "Organization ID (default: selected organization if logged in)")
	cmd.Flags().StringVar(&opts.Endpoint, "endpoint", "", "Client endpoint (required if not logged in)")
	cmd.Flags().IntVar(&opts.MTU, "mtu", 1280, "Maximum transmission unit")
	cmd.Flags().StringVar(&opts.DNS, "netstack-dns", defaultDNSServer, "DNS `server` to use for Netstack")
	cmd.Flags().StringVar(&opts.InterfaceName, "interface-name", "pangolin", "Interface `name`")
	cmd.Flags().StringVar(&opts.LogLevel, "log-level", "info", "Log level")
	cmd.Flags().StringVar(&opts.HTTPAddr, "http-addr", "", "HTTP address for API server")
	cmd.Flags().DurationVar(&opts.PingInterval, "ping-interval", 5*time.Second, "Ping `interval`")
	cmd.Flags().DurationVar(&opts.PingTimeout, "ping-timeout", 5*time.Second, "Ping `timeout`")
	cmd.Flags().BoolVar(&opts.Holepunch, "holepunch", true, "Enable holepunching")
	cmd.Flags().StringVar(&opts.TlsClientCert, "tls-client-cert", "", "TLS client certificate `path`")
	cmd.Flags().BoolVar(&opts.OverrideDNS, "override-dns", true, "Override system DNS for resolving internal resource alias")
	cmd.Flags().BoolVar(&opts.TunnelDNS, "tunnel-dns", false, "Use tunnel DNS for internal resource alias resolution")
	cmd.Flags().StringSliceVar(&opts.UpstreamDNS, "upstream-dns", []string{defaultDNSServer}, "List of DNS servers to use for external DNS resolution if overriding system DNS")
	cmd.Flags().BoolVar(&opts.Attached, "attach", false, "Run in attached (foreground) mode, (default: detached (background) mode)")
	cmd.Flags().BoolVar(&opts.Silent, "silent", false, "Disable TUI and run silently when detached")

	return cmd
}

func clientUpMain(cmd *cobra.Command, opts *ClientUpCmdOpts, extraArgs []string) error {
	apiClient := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())
	cfg := config.ConfigFromContext(cmd.Context())

	if runtime.GOOS == "windows" {
		err := errors.New("this command is currently unsupported on Windows")
		logger.Error("Error: %v", err)
		return err
	}

	// Check if a client is already running
	olmClient := olm.NewClient("")
	if olmClient.IsRunning() {
		err := errors.New("a client is already running")
		logger.Error("Error: %v", err)
		return err
	}

	// Use provided flags whenever possible.
	// No user session is needed when passing these directly,
	// so continue even if not logged in.
	olmID := opts.ID
	olmSecret := opts.Secret

	credentialsFromKeyring := olmID == "" && olmSecret == ""

	if credentialsFromKeyring {
		activeAccount, err := accountStore.ActiveAccount()
		if err != nil {
			logger.Error("Error: %v. Run `pangolin login` to login", err)
			return err
		}

		// Ensure OLM credentials exist and are valid
		newCredsGenerated, err := utils.EnsureOlmCredentials(apiClient, activeAccount)
		if err != nil {
			logger.Error("Failed to ensure OLM credentials: %v", err)
			return err
		}

		if newCredsGenerated {
			err := accountStore.Save()
			if err != nil {
				logger.Error("Failed to save accounts to store: %v", err)
				return err
			}
		}

		olmID = activeAccount.OlmCredentials.ID
		olmSecret = activeAccount.OlmCredentials.Secret
	}

	orgID := opts.OrgID

	// If no organization ID is specified, then use the active user's
	// selected organization if possible.
	if orgID == "" && credentialsFromKeyring {
		activeAccount, _ := accountStore.ActiveAccount()

		if activeAccount.OrgID == "" {
			err := errors.New("organization not selected")
			logger.Error("Error: %v", err)
			logger.Info("Run `pangolin select org` to select an organization or pass --org [id] to the command")
			return err
		}

		if err := utils.EnsureOrgAccess(apiClient, activeAccount); err != nil {
			logger.Error("%v", err)
			return err
		}

		orgID = activeAccount.OrgID
	}

	// Handle log file setup - if detached mode, always use log file
	var logFile string
	if !opts.Attached {
		logFile = cfg.LogFile
	}

	var endpoint string

	if opts.Endpoint == "" && credentialsFromKeyring {
		activeAccount, _ := accountStore.ActiveAccount()
		endpoint = activeAccount.Host
	} else {
		endpoint = opts.Endpoint
	}

	if endpoint == "" {
		err := errors.New("endpoint is required")
		logger.Error("Error: %v", err)
		logger.Info("Please login with a host or provide the --endpoint flag.")
		return err
	}

	// Handle detached mode - subprocess self without --attach flag
	// Skip detached mode if already running as root (we're a subprocess spawned by sudo)
	isRunningAsRoot := runtime.GOOS != "windows" && os.Geteuid() == 0
	if !opts.Attached && !isRunningAsRoot {
		executable, err := os.Executable()
		if err != nil {
			logger.Error("Error: failed to get executable path: %v", err)
			return err
		}

		// Build command arguments, excluding --attach flag
		cmdArgs := []string{"up", "client"}

		// Add org flag if needed (required for subprocess, which runs as
		// root and won't have user's config)
		if orgID != "" {
			cmdArgs = append(cmdArgs, "--org", orgID)
		}

		// Add all flags that were set (except --attach)
		// OLM credentials are always included (from flags, config, or newly created)
		cmdArgs = append(cmdArgs, "--id", olmID)
		cmdArgs = append(cmdArgs, "--secret", olmSecret)

		// Always pass endpoint to subprocess (required, subprocess won't have user's config)
		// Get endpoint from flag or hostname config (same logic as attached mode)
		cmdArgs = append(cmdArgs, "--endpoint", endpoint)

		// Optional flags - only include if they were explicitly set
		if cmd.Flags().Changed("mtu") {
			cmdArgs = append(cmdArgs, "--mtu", fmt.Sprintf("%d", opts.MTU))
		}
		if cmd.Flags().Changed("netstack-dns") {
			cmdArgs = append(cmdArgs, "--netstack-dns", opts.DNS)
		}
		if cmd.Flags().Changed("interface-name") {
			cmdArgs = append(cmdArgs, "--interface-name", opts.InterfaceName)
		}
		if cmd.Flags().Changed("log-level") {
			cmdArgs = append(cmdArgs, "--log-level", opts.LogLevel)
		}
		if cmd.Flags().Changed("http-addr") {
			cmdArgs = append(cmdArgs, "--http-addr", opts.HTTPAddr)
		}
		if cmd.Flags().Changed("ping-interval") {
			cmdArgs = append(cmdArgs, "--ping-interval", opts.PingInterval.String())
		}
		if cmd.Flags().Changed("ping-timeout") {
			cmdArgs = append(cmdArgs, "--ping-timeout", opts.PingTimeout.String())
		}
		if cmd.Flags().Changed("holepunch") {
			if opts.Holepunch {
				cmdArgs = append(cmdArgs, "--holepunch")
			} else {
				cmdArgs = append(cmdArgs, "--holepunch=false")
			}
		}
		if cmd.Flags().Changed("tls-client-cert") {
			cmdArgs = append(cmdArgs, "--tls-client-cert", opts.TlsClientCert)
		}
		if cmd.Flags().Changed("override-dns") {
			if opts.OverrideDNS {
				cmdArgs = append(cmdArgs, "--override-dns")
			} else {
				cmdArgs = append(cmdArgs, "--override-dns=false")
			}
		}
		if cmd.Flags().Changed("tunnel-dns") {
			if opts.TunnelDNS {
				cmdArgs = append(cmdArgs, "--tunnel-dns")
			} else {
				cmdArgs = append(cmdArgs, "--tunnel-dns=false")
			}
		}
		if cmd.Flags().Changed("upstream-dns") {
			// Comma sep
			cmdArgs = append(cmdArgs, "--upstream-dns", strings.Join(opts.UpstreamDNS, ","))
		}

		// Add positional args if any
		cmdArgs = append(cmdArgs, extraArgs...)

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
			err := errors.New("detached mode is not supported on Windows")
			logger.Error("Error: %v", err)
			return err
		}

		// Start the process
		if err := procCmd.Start(); err != nil {
			logger.Error("Error: failed to start detached process: %v", err)
			return err
		}

		// Wait for sudo to complete (password prompt + subprocess start)
		// The shell wrapper backgrounds the subprocess, so sudo exits immediately
		if err := procCmd.Wait(); err != nil {
			logger.Error("Error: failed to start subprocess: %v", err)
			return err
		}

		// In silent mode, skip TUI and just exit after starting the process
		if opts.Silent {
			return nil
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
					_, _ = client.Exit()
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
			logger.Error("Error: %v", err)
			return err
		}

		// Check if the process completed successfully or was killed
		if !completed {
			// User exited early - subprocess was killed
			logger.Info("Client process killed")
		} else {
			// Completed successfully
			logger.Success("Client interface created successfully")
		}
		return nil
	}

	enableAPI := defaultEnableAPI

	// In detached mode, API cannot be disabled (required for status/control)
	if !opts.Attached && !enableAPI {
		enableAPI = true
	}

	socketPath := defaultSocketPath

	upstreamDNS := make([]string, 0, len(opts.UpstreamDNS))
	for _, server := range opts.UpstreamDNS {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}

		if !strings.Contains(server, ":") {
			server = fmt.Sprintf("%s:53", server)
		}

		upstreamDNS = append(upstreamDNS, server)
	}

	// If no DNS servers were provided, force using
	// the default server again.
	if len(upstreamDNS) == 0 {
		upstreamDNS = []string{fmt.Sprintf("%s:53", defaultDNSServer)}
	}

	// Setup log file if specified
	if logFile != "" {
		if err := setupLogFile(cfg.LogFile); err != nil {
			logger.Error("Error: failed to setup log file: %v", err)
			return err
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
		activeAccount, err := accountStore.ActiveAccount()
		if err != nil {
			logger.Error("Failed to get session token: %v", err)
			return err
		}

		userToken = activeAccount.SessionToken
	}

	// Create context for signal handling and cleanup
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer olmpkg.Close()
	defer stop()

	// Create OLM GlobalConfig with hardcoded values from Swift
	olmInitConfig := olmpkg.GlobalConfig{
		LogLevel:   opts.LogLevel,
		EnableAPI:  enableAPI,
		SocketPath: socketPath,
		HTTPAddr:   opts.HTTPAddr,
		Version:    versionpkg.Version,
		Agent:      defaultAgent,
		OnTerminated: func() {
			logger.Info("Client process terminated")
			stop()
			os.Exit(0)
		},
		OnAuthError: func(statusCode int, message string) {
			logger.Error("Authentication error: %d %s", statusCode, message)
			stop()
			os.Exit(1)
		},
		OnExit: func() {
			logger.Info("Client process exiting")
			os.Exit(0)
		},
	}

	olmConfig := olmpkg.TunnelConfig{
		Endpoint:             endpoint,
		ID:                   olmID,
		Secret:               olmSecret,
		OrgID:                orgID,
		MTU:                  opts.MTU,
		DNS:                  opts.DNS,
		InterfaceName:        opts.InterfaceName,
		Holepunch:            opts.Holepunch,
		TlsClientCert:        opts.TlsClientCert,
		PingIntervalDuration: opts.PingInterval,
		PingTimeoutDuration:  opts.PingTimeout,
		OverrideDNS:          opts.OverrideDNS,
		TunnelDNS:            opts.TunnelDNS,
		UpstreamDNS:          upstreamDNS,
	}

	// Add UserToken if we have it (from flag or config)
	if userToken != "" {
		olmConfig.UserToken = userToken
	}

	// Check if running with elevated permissions (required for network interface creation)
	// This check is only for attached mode; in detached mode, the subprocess runs elevated
	if runtime.GOOS != "windows" {
		if os.Geteuid() != 0 {
			err := errors.New("elevated permissions are required for network interface creation")
			logger.Error("Error: %v", err)
			logger.Info("Please run with sudo or use detached mode (default) to run the subprocess elevated.")
			return err
		}
	}

	olmpkg.Init(ctx, olmInitConfig)
	if enableAPI {
		_ = olmpkg.StartApi()
	}
	olmpkg.StartTunnel(olmConfig)

	return nil
}

// setupLogFile sets up file logging with rotation
func setupLogFile(logPath string) error {
	logDir := filepath.Dir(logPath)

	// Rotate log file if needed
	err := rotateLogFile(logDir, logPath)
	if err != nil {
		// Log warning but continue
		log.Printf("Warning: failed to rotate log file: %v", err)
	}

	// Open log file for appending
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// Set the logger output
	newtLogger.GetLogger().SetOutput(file)

	// log.Printf("Logging to file: %s", logPath)
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
				_ = os.Remove(filePath)
			}
		}
	}
}
