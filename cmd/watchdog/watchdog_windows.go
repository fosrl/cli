//go:build windows

package watchdog

import "github.com/spf13/cobra"

// WatchdogCmd is unsupported on Windows.
func WatchdogCmd() *cobra.Command {
	return nil
}
