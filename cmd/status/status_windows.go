//go:build windows

package status

import (
	"github.com/spf13/cobra"
)

// StatusCmd returns nil on Windows as this command is not supported.
func StatusCmd() *cobra.Command {
	return nil
}