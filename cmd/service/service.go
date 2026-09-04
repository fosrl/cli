// Package service implements `pangolin service`, which installs, removes,
// and monitors systemd units that keep `pangolin up site` or
// `pangolin up client` running persistently in the background - restarting
// them automatically if they crash or the machine reboots. This mirrors the
// "Systemd Service" install instructions shown for the standalone newt/olm
// binaries, but driven from the CLI itself.
package service

import "github.com/spf13/cobra"

// ServiceCmd returns the `service` command tree:
//
//	pangolin service install site|client
//	pangolin service uninstall site|client
//	pangolin service status site|client
//	pangolin service logs site|client
func ServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage systemd services that keep a site or client running persistently",
		Long: `Install, remove, and monitor systemd services that run 'pangolin up site' or
'pangolin up client' in the background, restarting them automatically if they
crash or the machine reboots.

Only supported on Linux (requires systemd) and must be run as root.`,
	}

	cmd.AddCommand(installCmd())
	cmd.AddCommand(uninstallCmd())
	cmd.AddCommand(statusCmd())
	cmd.AddCommand(logsCmd())

	return cmd
}

func installCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and start a systemd service",
	}

	cmd.AddCommand(siteInstallCmd())
	cmd.AddCommand(clientInstallCmd())

	return cmd
}

func uninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Stop and remove a systemd service",
	}

	cmd.AddCommand(siteUninstallCmd())
	cmd.AddCommand(clientUninstallCmd())

	return cmd
}

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show a systemd service's status",
	}

	cmd.AddCommand(siteStatusCmd())
	cmd.AddCommand(clientStatusCmd())

	return cmd
}

func logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Follow a systemd service's logs",
	}

	cmd.AddCommand(siteLogsCmd())
	cmd.AddCommand(clientLogsCmd())

	return cmd
}
