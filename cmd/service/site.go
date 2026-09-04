package service

import (
	"fmt"
	"os"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/systemdsvc"
	"github.com/spf13/cobra"
)

// siteServiceName is the systemd unit name (without the .service suffix)
// used to run a site tunnel persistently in the background.
const siteServiceName = "pangolin-site"

func siteInstallCmd() *cobra.Command {
	opts := struct {
		ID             string
		Secret         string
		Endpoint       string
		DisableClients bool
		DisableSSH     bool
	}{}

	cmd := &cobra.Command{
		Use:   "site",
		Short: "Install and start the site (Newt) systemd service",
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
				Name:        siteServiceName,
				Description: "Pangolin Site (Newt)",
				ExecStart:   fmt.Sprintf("%s up site", executable),
				EnvVars:     envVars,
			}

			if err := systemdsvc.Install(spec); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}

			logger.Success("Installed and started %s.service", siteServiceName)
			logger.Info("Check status with 'pangolin service status site' or follow logs with 'pangolin service logs site -f'")
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

func siteUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "site",
		Short: "Stop and remove the site (Newt) systemd service",
		Run: func(cmd *cobra.Command, args []string) {
			if err := systemdsvc.Uninstall(siteServiceName); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
			logger.Success("Removed %s.service", siteServiceName)
		},
	}
}

func siteStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "site",
		Short: "Show the site (Newt) systemd service status",
		Run: func(cmd *cobra.Command, args []string) {
			out, err := systemdsvc.Status(siteServiceName)
			if err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
			fmt.Print(out)
		},
	}
}

func siteLogsCmd() *cobra.Command {
	opts := struct{ Lines int }{}

	cmd := &cobra.Command{
		Use:   "site",
		Short: "Follow the site (Newt) systemd service logs",
		Long:  "Stream the site service's journal output (equivalent to 'journalctl -u pangolin-site -f').",
		Run: func(cmd *cobra.Command, args []string) {
			if err := systemdsvc.Follow(siteServiceName, opts.Lines); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().IntVarP(&opts.Lines, "lines", "n", 20, "Number of prior lines to show before following")

	return cmd
}
