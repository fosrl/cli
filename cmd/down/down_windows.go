//go:build windows

package down

import (
	"github.com/spf13/cobra"
)

// DownCmd returns nil on Windows as this command is not supported.
func DownCmd() *cobra.Command {
	return nil
}