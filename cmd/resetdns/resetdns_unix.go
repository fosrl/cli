//go:build !windows

package resetdns

import (
	"errors"
	"os"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	dnsOverride "github.com/fosrl/olm/dns/override"
	"github.com/spf13/cobra"
)

// ResetDNSCmd returns the `pangolin reset-dns` command which forcibly
// removes any stale DNS override left behind by a crashed client.
func ResetDNSCmd() *cobra.Command {
	var interfaceName string
	var force bool

	cmd := &cobra.Command{
		Use:   "reset-dns",
		Short: "Force-clear stale DNS overrides",
		Long: `Forcibly clear stale DNS overrides left behind by a crashed or
stuck client. This restores your system DNS to its original
configuration.

By default this command refuses to run when a client is still
active; use --force to override that check.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := olm.NewClient("")
			if client.IsRunning() && !force {
				return errors.New("a client is currently running; stop it first with 'pangolin down' or rerun with --force")
			}
			if client.IsRunning() && force {
				logger.Warning("Client appears to still be running; attempting reset anyway because --force was passed")
			}

			if os.Geteuid() != 0 {
				logger.Warning("DNS reset typically requires root privileges; rerun with sudo if it fails")
			}

			if err := dnsOverride.ForceResetDNS(interfaceName); err != nil {
				logger.Error("DNS reset failed: %v", err)
				return err
			}

			logger.Success("DNS configuration reset")
			return nil
		},
	}

	cmd.Flags().StringVar(&interfaceName, "interface", "pangolin", "Tunnel interface name to clean up")
	cmd.Flags().BoolVar(&force, "force", false, "Run the reset even if a client appears to be active")

	return cmd
}
