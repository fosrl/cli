package service

import (
	"fmt"
	"os"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/svcmgr"
	"github.com/spf13/cobra"
)

// clientServiceName identifies the background service (systemd unit,
// launchd label, or Windows Service name, depending on platform) used to
// run a machine client persistently.
const clientServiceName = svcmgr.ClientServiceName

// The client subcommands target machine clients (explicit --id/--secret,
// no user login) since that's the case that needs to run unattended on a
// server; an interactively-logged-in user's client is expected to be
// started manually.

func clientInstallCmd() *cobra.Command {
	opts := struct {
		ID       string
		Secret   string
		Endpoint string
		OrgID    string
	}{}

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Install and start the client background service",
		Long: `Install a background service for this machine client, then start it
immediately.

Intended for machine clients (--id/--secret), which don't have an
interactively logged-in user to restart them. On Windows this only supports
the service itself - 'pangolin up client' has no standalone Windows console
mode (that's handled by the Pangolin desktop app); the service runs the
tunnel directly in-process instead.`,
		Run: func(cmd *cobra.Command, args []string) {
			envVars := map[string]string{
				"PANGOLIN_CLIENT_ID":     opts.ID,
				"PANGOLIN_CLIENT_SECRET": opts.Secret,
				"PANGOLIN_ENDPOINT":      opts.Endpoint,
			}
			if opts.OrgID != "" {
				envVars["PANGOLIN_ORG"] = opts.OrgID
			}

			spec := svcmgr.Spec{
				Name:        clientServiceName,
				DisplayName: "Pangolin Client",
				Description: "Runs 'pangolin up client' persistently in the background",
				// On Linux/macOS, Args is the subprocess command line
				// (--attach runs in the foreground under the service
				// supervisor's control, no self-detaching subprocess/TUI).
				// Windows ignores Args for the client service and instead
				// runs the tunnel in-process (see svcmgr_windows.go) - it's
				// kept here for documentation/parity. Credentials always
				// come from the environment rather than the command line so
				// they don't leak into `ps`/Task Manager.
				Args:    []string{"up", "client", "--attach"},
				EnvVars: envVars,
			}

			if err := svcmgr.Install(spec); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}

			logger.Success("Installed and started the %s service", clientServiceName)
			logger.Info("Check status with 'pangolin service status client' or follow logs with 'pangolin service logs client -f'")
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "Client ID")
	cmd.Flags().StringVar(&opts.Secret, "secret", "", "Client secret")
	cmd.Flags().StringVar(&opts.Endpoint, "endpoint", "", "Pangolin server endpoint")
	cmd.Flags().StringVar(&opts.OrgID, "org", "", "Organization ID")
	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("secret")
	cmd.MarkFlagRequired("endpoint")

	return cmd
}

func clientUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "client",
		Short: "Stop and remove the client background service",
		Run: func(cmd *cobra.Command, args []string) {
			if err := svcmgr.Uninstall(clientServiceName); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
			logger.Success("Removed the %s service", clientServiceName)
		},
	}
}

func clientStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "client",
		Short: "Show the client background service status",
		Run: func(cmd *cobra.Command, args []string) {
			out, err := svcmgr.Status(clientServiceName)
			if err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
			fmt.Print(out)
		},
	}
}

func clientLogsCmd() *cobra.Command {
	opts := struct{ Lines int }{}

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Follow the client background service logs",
		Long:  "Stream the client service's log output.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := svcmgr.Follow(clientServiceName, opts.Lines); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().IntVarP(&opts.Lines, "lines", "n", 20, "Number of prior lines to show before following")

	return cmd
}
