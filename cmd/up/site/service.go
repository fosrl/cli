package site

import (
	"fmt"
	"os"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/systemdsvc"
	"github.com/spf13/cobra"
)

// serviceName is the systemd unit name (without the .service suffix) used to
// run this site tunnel persistently in the background.
const serviceName = "pangolin-site"

// ServiceCmd returns the `up site service` command group, which installs,
// removes, and monitors a systemd service that keeps `pangolin up site`
// running persistently - restarting it automatically on crash or reboot.
func ServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage a systemd service that keeps this site tunnel running persistently",
		Long: `Install, remove, and monitor a systemd service that runs 'pangolin up site'
in the background, restarting it automatically if it crashes or the machine
reboots.

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
		ID             string
		Secret         string
		Endpoint       string
		DisableClients bool
		DisableSSH     bool
	}{}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and start the systemd service",
		Long:  "Write a systemd unit and environment file for this site, then enable and start it immediately.",
		Run: func(cmd *cobra.Command, args []string) {
			executable, err := os.Executable()
			if err != nil {
				logger.Error("Error: failed to resolve executable path: %v", err)
				os.Exit(1)
			}

			envVars := map[string]string{
				"NEWT_ID":           opts.ID,
				"NEWT_SECRET":       opts.Secret,
				"PANGOLIN_ENDPOINT": opts.Endpoint,
			}
			if opts.DisableClients {
				envVars["DISABLE_CLIENTS"] = "true"
			}
			if opts.DisableSSH {
				envVars["DISABLE_SSH"] = "true"
			}

			spec := systemdsvc.UnitSpec{
				Name:        serviceName,
				Description: "Pangolin Site (Newt)",
				ExecStart:   fmt.Sprintf("%s up site", executable),
				EnvVars:     envVars,
			}

			if err := systemdsvc.Install(spec); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}

			logger.Success("Installed and started %s.service", serviceName)
			logger.Info("Check status with 'pangolin up site service status' or follow logs with 'pangolin up site service logs -f'")
		},
	}

	cmd.Flags().StringVar(&opts.ID, "id", "", "Site ID")
	cmd.Flags().StringVar(&opts.Secret, "secret", "", "Site secret")
	cmd.Flags().StringVar(&opts.Endpoint, "endpoint", "", "Pangolin server endpoint")
	cmd.Flags().BoolVar(&opts.DisableClients, "disable-clients", false, "Disable accepting client connections")
	cmd.Flags().BoolVar(&opts.DisableSSH, "disable-ssh", false, "Disable Pangolin SSH")
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
			if err := systemdsvc.Uninstall(serviceName); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
			logger.Success("Removed %s.service", serviceName)
		},
	}
}

func serviceStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the systemd service status",
		Run: func(cmd *cobra.Command, args []string) {
			out, err := systemdsvc.Status(serviceName)
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
		Long:  "Stream the site service's journal output (equivalent to 'journalctl -u pangolin-site -f').",
		Run: func(cmd *cobra.Command, args []string) {
			if err := systemdsvc.Follow(serviceName, opts.Lines); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().IntVarP(&opts.Lines, "lines", "n", 20, "Number of prior lines to show before following")

	return cmd
}
