//go:build darwin

package authdaemon

import (
	"github.com/spf13/cobra"
)

// AuthDaemonCmd returns nil on macOS as this command is not supported.
func AuthDaemonCmd() *cobra.Command {
	return nil
}
