//go:build windows

package authdaemon

import (
	"github.com/spf13/cobra"
)

// AuthDaemonCmd returns nil on Windows as this command is not supported.
func AuthDaemonCmd() *cobra.Command {
	return nil
}
