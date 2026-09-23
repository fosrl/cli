package service

import (
	"fmt"
	"os"
	"path/filepath"

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
		ConfigFile     string
		DisableClients bool
		DisableSSH     bool
	}{}

	cmd := &cobra.Command{
		Use:   "site",
		Short: "Install and start the site (Newt) background service",
		Long: `Install a background service for this site, then start it immediately.

Credentials can be given directly (--id, --secret, --endpoint) or via a
newt config file (--config-file), in which case the flags are optional and
any that are set override the file's values.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.ConfigFile != "" {
				abs, err := filepath.Abs(opts.ConfigFile)
				if err != nil {
					return fmt.Errorf("resolving --config-file: %w", err)
				}
				if _, err := os.Stat(abs); err != nil {
					return fmt.Errorf("--config-file: %w", err)
				}
				opts.ConfigFile = abs
				return nil
			}
			var missing []string
			for _, f := range []struct{ name, val string }{
				{"id", opts.ID}, {"secret", opts.Secret}, {"endpoint", opts.Endpoint},
			} {
				if f.val == "" {
					missing = append(missing, "--"+f.name)
				}
			}
			if len(missing) > 0 {
				return fmt.Errorf("required flag(s) %v not set (or pass --config-file)", missing)
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			envVars := map[string]string{}
			for k, v := range map[string]string{
				"NEWT_ID":           opts.ID,
				"NEWT_SECRET":       opts.Secret,
				"PANGOLIN_ENDPOINT": opts.Endpoint,
				"CONFIG_FILE":       opts.ConfigFile,
			} {
				if v != "" {
					envVars[k] = v
				}
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

	cmd.Flags().StringVar(&opts.ID, "id", "", "Site ID (required unless --config-file is set)")
	cmd.Flags().StringVar(&opts.Secret, "secret", "", "Site secret (required unless --config-file is set)")
	cmd.Flags().StringVar(&opts.Endpoint, "endpoint", "", "Pangolin server endpoint (required unless --config-file is set)")
	cmd.Flags().StringVar(&opts.ConfigFile, "config-file", "", "Path to a newt config file passed through to 'pangolin up site'")
	cmd.Flags().BoolVar(&opts.DisableClients, "disable-clients", false, "Disable accepting client connections")
	cmd.Flags().BoolVar(&opts.DisableSSH, "disable-ssh", false, "Disable Pangolin SSH")

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
