package service

import (
	"fmt"
	"os"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/svcmgr"
	"github.com/spf13/cobra"
)

// siteServiceName identifies the background service (systemd unit, launchd
// label, or Windows Service name, depending on platform) used to run a
// site tunnel persistently.
const siteServiceName = svcmgr.SiteServiceName

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
		Short: "Install and start the site (Newt) background service",
		Long:  "Install a background service for this site, then start it immediately.",
		Run: func(cmd *cobra.Command, args []string) {
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

			spec := svcmgr.Spec{
				Name:        siteServiceName,
				DisplayName: "Pangolin Site (Newt)",
				Description: "Runs 'pangolin up site' persistently in the background",
				Args:        []string{"up", "site"},
				EnvVars:     envVars,
			}

			if err := svcmgr.Install(spec); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}

			logger.Success("Installed and started the %s service", siteServiceName)
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
		Short: "Stop and remove the site (Newt) background service",
		Run: func(cmd *cobra.Command, args []string) {
			if err := svcmgr.Uninstall(siteServiceName); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
			logger.Success("Removed the %s service", siteServiceName)
		},
	}
}

func siteStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "site",
		Short: "Show the site (Newt) background service status",
		Run: func(cmd *cobra.Command, args []string) {
			out, err := svcmgr.Status(siteServiceName)
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
		Short: "Follow the site (Newt) background service logs",
		Long:  "Stream the site service's log output.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := svcmgr.Follow(siteServiceName, opts.Lines); err != nil {
				logger.Error("Error: %v", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().IntVarP(&opts.Lines, "lines", "n", 20, "Number of prior lines to show before following")

	return cmd
}
