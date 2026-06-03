//go:build !windows

package cleanup

import (
	"errors"
	"os"
	"runtime"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olmdns"
	"github.com/spf13/cobra"
)

func CleanupCmd() *cobra.Command {
	interfaceName := olmdns.DefaultInterfaceName

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up stale DNS configuration",
		Long: `Remove stale DNS configuration left from an unclean shutdown.

This is useful if the client was killed while a tunnel was active and system DNS
was not restored. The same cleanup runs automatically before starting a connection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "windows" && os.Geteuid() != 0 {
				return errors.New("elevated permissions are required to clean up DNS configuration")
			}

			if err := olmdns.CleanupStaleState(interfaceName); err != nil {
				logger.Error("Failed to clean up stale DNS configuration: %v", err)
				return err
			}

			logger.Success("Stale DNS configuration cleaned up")
			return nil
		},
	}

	cmd.Flags().StringVar(&interfaceName, "interface-name", olmdns.DefaultInterfaceName, "WireGuard interface `name` used when the tunnel was active")

	return cmd
}
