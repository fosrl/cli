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
	"github.com/fosrl/cli/cmd/down"
	"github.com/fosrl/cli/cmd/logs"
	selectcmd "github.com/fosrl/cli/cmd/select"
	"github.com/fosrl/cli/cmd/ssh"
	"github.com/fosrl/cli/cmd/status"
	"github.com/fosrl/cli/cmd/up"
	"github.com/fosrl/cli/cmd/update"
	"github.com/fosrl/cli/cmd/version"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
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
	cmd.AddCommand(selectcmd.SelectCmd())

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

	cmd.AddCommand(ssh.SSHCmd())
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

	accountStore, err := config.LoadAccountStore()
	if err != nil {
		return nil, err
	}

	var apiBaseURL string
	var sessionToken string

	if activeAccount, _ := accountStore.ActiveAccount(); activeAccount != nil {
		apiBaseURL = activeAccount.Host
		sessionToken = activeAccount.SessionToken
	} else {
		apiBaseURL = ""
		sessionToken = ""
	}

	client, err := api.InitClient(apiBaseURL, sessionToken)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ctx = api.WithAPIClient(ctx, client)
	ctx = config.WithAccountStore(ctx, accountStore)
	ctx = config.WithConfig(ctx, cfg)

	cmd.SetContext(ctx)

	return cmd, nil
}

func mainCommandPreRun(cmd *cobra.Command, args []string) error {
	if shouldSkipRuntimeInit(cmd) {
		return nil
	}

	cfg := config.ConfigFromContext(cmd.Context())

	if err := ensureRuntimeDirs(cfg); err != nil {
		return err
	}

	// Check for updates asynchronously
	if !cfg.DisableUpdateCheck {
		versionpkg.CheckForUpdateAsync(func(release *versionpkg.GitHubRelease) {
			logger.Warning("A new version is available: %s (current: %s)", release.TagName, versionpkg.Version)
			logger.Info("Run 'pangolin update' to update to the latest version")
			fmt.Println()
		})
	}

	return nil
}

// shouldSkipRuntimeInit returns true for commands that must not touch runtime
// directories or emit diagnostics to stdout (for example shell completion).
func shouldSkipRuntimeInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "completion", "version", "update":
			return true
		}
	}
	return false
}

// Make sure all required directories exist once before executing subcommands.
func ensureRuntimeDirs(cfg *config.Config) error {
	configDir, err := config.GetPangolinConfigDir()
	if err != nil {
		return fmt.Errorf("failed to create pangolin configuration directory: %w", err)
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", configDir, err)
	}

	if cfg.LogFile != "" {
		logPathDirname := filepath.Dir(cfg.LogFile)
		if err := os.MkdirAll(logPathDirname, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", logPathDirname, err)
		}
	}

	return nil
}

// Execute is called by main.go
func Execute() {
	cmd, err := RootCommand(true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
