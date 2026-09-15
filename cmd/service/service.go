// Package service implements `pangolin service`, which installs, removes,
// and monitors background services that keep `pangolin up site` or
// `pangolin up client` running persistently - restarting them automatically
// if they crash or the machine reboots. This mirrors the "Systemd Service"
// install instructions shown for the standalone newt/olm binaries, but
// driven from the CLI itself and backed by systemd, launchd, or a Windows
// Service depending on platform (see internal/svcmgr).
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
		Short: "Manage background services that keep a site or client running persistently",
		Long: `Install, remove, and monitor background services that run 'pangolin up site'
or 'pangolin up client' persistently, restarting them automatically if they
crash or the machine reboots.

Backed by systemd on Linux, launchd on macOS, and a Windows Service on
Windows. Must be run as root (Linux/macOS) or from an elevated prompt
(Windows). The client (machine client) service isn't available on Windows
yet, since 'pangolin up client' itself doesn't support Windows.`,
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
		Short: "Install and start a background service",
	}

	cmd.AddCommand(siteInstallCmd())
	cmd.AddCommand(clientInstallCmd())

	return cmd
}

func uninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Stop and remove a background service",
	}

	cmd.AddCommand(siteUninstallCmd())
	cmd.AddCommand(clientUninstallCmd())

	return cmd
}

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show a background service's status",
	}

	cmd.AddCommand(siteStatusCmd())
	cmd.AddCommand(clientStatusCmd())

	return cmd
}

func logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Follow a background service's logs",
	}

	cmd.AddCommand(siteLogsCmd())
	cmd.AddCommand(clientLogsCmd())

	return cmd
}
