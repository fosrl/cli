package client

import (
	"fmt"
	"os"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/systemdsvc"
	"github.com/spf13/cobra"
)

// clientServiceName is the systemd unit name (without the .service suffix)
// used to run this machine client persistently in the background.
const clientServiceName = "pangolin-client"

// ServiceCmd returns the `up client service` command group, which installs,
// removes, and monitors a systemd service that keeps `pangolin up client`
// running persistently - restarting it automatically on crash or reboot.
//
// This targets machine clients (explicit --id/--secret, no user login) since
// that's the case that needs to run unattended on a server; an
// interactively-logged-in user's client is expected to be started manually.
func ServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage a systemd service that keeps this machine client running persistently",
		Long: `Install, remove, and monitor a systemd service that runs 'pangolin up client'
in the background, restarting it automatically if it crashes or the machine
reboots.

Intended for machine clients (--id/--secret), which don't have an
interactively logged-in user to restart them.

Only supported on Linux (requires systemd) and must be run as root.`,
	}

	cmd.AddCommand(serviceInstallCmd())
	cmd.AddCommand(serviceUninstallCmd())
	cmd.AddCommand(serviceStatusCmd())
	cmd.AddCommand(serviceLogsCmd())

	return cmd
}

func serviceInstallCmd() *cobra.Command {
	opts := struct {
		ID       string
		Secret   string
		Endpoint string
		OrgID    string
	}{}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and start the systemd service",
		Long:  "Write a systemd unit and environment file for this client, then enable and start it immediately.",
		Run: func(cmd *cobra.Command, args []string) {
			executable, err := os.Executable()
			if err != nil {
				logger.Error("Error: failed to resolve executable path: %v", err)
				os.Exit(1)
			}

			envVars := map[string]string{
				"PANGOLIN_CLIENT_ID":     opts.ID,
				"PANGOLIN_CLIENT_SECRET": opts.Secret,
				"PANGOLIN_ENDPOINT":      opts.Endpoint,
			}
			if opts.OrgID != "" {
				envVars["PANGOLIN_ORG"] = opts.OrgID
			}

			spec := systemdsvc.UnitSpec{
				Name:        clientServiceName,
				Description: "Pangolin Client (Olm)",
				// --attach runs in the foreground under systemd's supervision
				// (no self-detaching subprocess/TUI); credentials come from
				// the EnvironmentFile rather than the command line so they
				// don't leak into `ps` output.
				ExecStart: fmt.Sprintf("%s up client --attach", executable),
				EnvVars:   envVars,
			}

			if err := systemdsvc.Install(spec); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}

			logger.Success("Installed and started %s.service", clientServiceName)
			logger.Info("Check status with 'pangolin up client service status' or follow logs with 'pangolin up client service logs -f'")
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

func serviceUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Stop and remove the systemd service",
		Run: func(cmd *cobra.Command, args []string) {
			if err := systemdsvc.Uninstall(clientServiceName); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
			logger.Success("Removed %s.service", clientServiceName)
		},
	}
}

func serviceStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the systemd service status",
		Run: func(cmd *cobra.Command, args []string) {
			out, err := systemdsvc.Status(clientServiceName)
			if err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
			fmt.Print(out)
		},
	}
}

func serviceLogsCmd() *cobra.Command {
	opts := struct{ Lines int }{}

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Follow the systemd service logs",
		Long:  "Stream the client service's journal output (equivalent to 'journalctl -u pangolin-client -f').",
		Run: func(cmd *cobra.Command, args []string) {
			if err := systemdsvc.Follow(clientServiceName, opts.Lines); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().IntVarP(&opts.Lines, "lines", "n", 20, "Number of prior lines to show before following")

	return cmd
}
