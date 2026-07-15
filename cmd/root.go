package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fosrl/cli/cmd/apply"
	"github.com/fosrl/cli/cmd/auth"
	"github.com/fosrl/cli/cmd/auth/login"
	"github.com/fosrl/cli/cmd/auth/logout"
	"github.com/fosrl/cli/cmd/authdaemon"
	companioncmd "github.com/fosrl/cli/cmd/companion"
	configcmd "github.com/fosrl/cli/cmd/config"
	"github.com/fosrl/cli/cmd/down"
	"github.com/fosrl/cli/cmd/list"
	"github.com/fosrl/cli/cmd/logs"
	"github.com/fosrl/cli/cmd/resetdns"
	"github.com/fosrl/cli/cmd/scp"
	selectcmd "github.com/fosrl/cli/cmd/select"
	"github.com/fosrl/cli/cmd/ssh"
	"github.com/fosrl/cli/cmd/status"
	"github.com/fosrl/cli/cmd/up"
	"github.com/fosrl/cli/cmd/update"
	"github.com/fosrl/cli/cmd/version"
	"github.com/fosrl/cli/cmd/watchdog"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/companion"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/notice"
	versionpkg "github.com/fosrl/cli/internal/version"
	"github.com/spf13/cobra"
)

// Initialize a root Cobra command.
//
// Set initResources to false when generating documentation to avoid
// parsing configuration files and instantiating the API client, among
// other such external resources. This is to avoid depending on external
// state when doing doc generation.
func RootCommand(initResources bool) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:          "pangolin",
		Short:        "Pangolin CLI",
		SilenceUsage: true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		PersistentPreRunE: mainCommandPreRun,
	}

	cmd.AddCommand(auth.AuthCommand())
	if authDaemonCmd := authdaemon.AuthDaemonCmd(); authDaemonCmd != nil {
		cmd.AddCommand(authDaemonCmd)
	}
	cmd.AddCommand(apply.ApplyCommand())
	if companionCmd := companioncmd.CompanionCmd(); companionCmd != nil {
		cmd.AddCommand(companionCmd)
	}
	cmd.AddCommand(selectcmd.SelectCmd())
	cmd.AddCommand(list.ListCmd())
	cmd.AddCommand(configcmd.ConfigCmd())

	// Platform-specific commands - nil on unsupported platforms
	if upCmd := up.UpCmd(); upCmd != nil {
		cmd.AddCommand(upCmd)
	}
	if downCmd := down.DownCmd(); downCmd != nil {
		cmd.AddCommand(downCmd)
	}
	if logsCmd := logs.LogsCmd(); logsCmd != nil {
		cmd.AddCommand(logsCmd)
	}
	if statusCmd := status.StatusCmd(); statusCmd != nil {
		cmd.AddCommand(statusCmd)
	}
	if resetDNSCmd := resetdns.ResetDNSCmd(); resetDNSCmd != nil {
		cmd.AddCommand(resetDNSCmd)
	}
	if watchdogCmd := watchdog.WatchdogCmd(); watchdogCmd != nil {
		cmd.AddCommand(watchdogCmd)
	}

	cmd.AddCommand(ssh.SSHCmd())
	cmd.AddCommand(scp.SCPCmd())
	cmd.AddCommand(update.UpdateCmd())
	cmd.AddCommand(version.VersionCmd())
	cmd.AddCommand(login.LoginCmd())
	cmd.AddCommand(logout.LogoutCmd())

	if !initResources {
		return cmd, nil
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	logger.InitLogger(cfg.LogLevel)

	ctx := context.Background()
	ctx = config.WithConfig(ctx, cfg)
	cmd.SetContext(ctx)

	return cmd, nil
}

func commandNeedsAuthInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "companion" || c.Name() == "config" {
			return false
		}
	}
	return true
}

func commandNeedsCompanionReady(cmd *cobra.Command) bool {
	if !commandNeedsAuthInit(cmd) {
		return false
	}
	return !commandExemptFromCompanionReady(cmd)
}

func commandExemptFromCompanionReady(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "version", "update", "login", "logout", "account", "org":
		return true
	}

	if cmd.Name() == "status" && commandHasAncestor(cmd, "auth") {
		return true
	}

	return false
}

func commandHasAncestor(cmd *cobra.Command, name string) bool {
	for p := cmd.Parent(); p != nil; p = p.Parent() {
		if p.Name() == name {
			return true
		}
	}
	return false
}

func initAuthContext(ctx context.Context, cfg *config.Config) (context.Context, error) {
	if _, ok := api.ClientFromContext(ctx); ok {
		return ctx, nil
	}

	companionState, accountStore, err := companion.Resolve(cfg)
	if err != nil {
		return ctx, err
	}

	var apiBaseURL string
	var sessionToken string

	if activeAccount, _ := accountStore.ActiveAccount(); activeAccount != nil {
		apiBaseURL = activeAccount.Host
		sessionToken = activeAccount.SessionToken
	}

	client, err := api.InitClient(apiBaseURL, sessionToken)
	if err != nil {
		return ctx, err
	}

	ctx = api.WithAPIClient(ctx, client)
	ctx = config.WithAccountStore(ctx, accountStore)
	ctx = companion.WithState(ctx, companionState)
	return ctx, nil
}

func mainCommandPreRun(cmd *cobra.Command, args []string) error {
	cfg := config.ConfigFromContext(cmd.Context())
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	if err := notice.ShowPending(cfg); err != nil {
		logger.Debug("Failed to show pending notices: %v", err)
	}

	if commandNeedsAuthInit(cmd) {
		ctx, err := initAuthContext(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		cmd.SetContext(ctx)
		cfg = config.ConfigFromContext(ctx)

		if commandNeedsCompanionReady(cmd) {
			if err := companion.GuardReady(ctx); err != nil {
				return err
			}
		}
	}

	// Skip update checks when running self-update or companion commands.
	cmdName := cmd.Name()
	if cmdName == "update" || !commandNeedsAuthInit(cmd) {
		logger.Debug("Skipping update check for %q command", cmdName)
		return nil
	}

	ensureRuntimeDirs(cfg)

	// Check for updates asynchronously
	if !cfg.DisableUpdateCheck {
		logger.Debug("Starting update check for %q command", cmdName)
		versionpkg.CheckForUpdateAsync(func(release *versionpkg.GitHubRelease) {
			logger.Warning("A new version is available: %s (current: %s)", release.TagName, versionpkg.Version)
			logger.Info("Run 'pangolin update' to update to the latest version")
			fmt.Println()
		})
	} else {
		logger.Debug("Update check disabled by configuration")
	}

	return nil
}

// Make sure all required directories exist once
// before executing any subcommands.
func ensureRuntimeDirs(cfg *config.Config) {
	configDir, err := config.GetPangolinConfigDir()
	if err != nil {
		logger.Warning("failed to create pangolin configuration directory: %v", err)
	} else {
		err = os.MkdirAll(configDir, 0o755)
		if err != nil {
			logger.Warning("failed to create %s: %v", configDir, err)
		}
	}

	if cfg.LogFile != "" {
		logPathDirname := filepath.Dir(cfg.LogFile)

		err = os.MkdirAll(logPathDirname, 0o755)
		if err != nil {
			logger.Warning("failed to create %s: %v", logPathDirname, err)
		}
	}
}

// Execute is called by main.go
func Execute() {
	cmd, err := RootCommand(true)
	if err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
