// Package site implements `pangolin up site`, which runs a Newt site
// tunnel embedded directly in the Pangolin CLI. It accepts exactly the same
// command-line flags and environment variables as the standalone newt
// binary (see github.com/fosrl/newt), reusing newt's own config-loading and
// runtime packages rather than re-implementing them.
package site

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/fosrl/cli/internal/logger"
	versionpkg "github.com/fosrl/cli/internal/version"
	"github.com/fosrl/newt/clients/permissions"
	newtLogger "github.com/fosrl/newt/logger"
	newtpkg "github.com/fosrl/newt/newt"
	"github.com/fosrl/newt/newtconfig"
	"github.com/spf13/cobra"
)

// SiteUpCmd returns the `up site` command. Flag parsing is delegated
// entirely to newtconfig (the same flag set the newt binary itself parses),
// so this command intentionally does not declare any cobra flags of its
// own - run `pangolin up site --help` to see them.
func SiteUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "site",
		Short: "Start a site connection (Newt)",
		Long: `Bring up a site tunnel using Newt, embedded directly in the Pangolin CLI.

This accepts the same flags and environment variables as the standalone
newt binary. Run 'pangolin up site --help' to see them.`,
		DisableFlagParsing: true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), args)
		},
	}

	return cmd
}

func run(ctx context.Context, args []string) error {
	newtLogger.Init(nil)

	cfg, err := newtconfig.Load(newtconfig.Options{
		Args:     args,
		Version:  versionpkg.NewtVersion(),
		Platform: runtime.GOOS,
	})
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	if cfg.UseNativeMainInterface {
		if err := permissions.CheckNativeInterfacePermissions(); err != nil {
			return fmt.Errorf("insufficient permissions for native main tunnel interface: %w", err)
		}
	}

	// Restart in place (preserving the original "up site" arguments)
	// rather than exiting, matching the standalone newt binary's behavior
	// when a blueprint reload requires a fresh tunnel.
	cfg.OnRestart = reexec

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	n, err := newtpkg.Init(sigCtx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize newt: %w", err)
	}

	logger.Info("Starting site tunnel to %s", cfg.Endpoint)
	n.Start(sigCtx)

	return nil
}
