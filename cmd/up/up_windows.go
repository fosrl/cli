//go:build windows

package up

import (
	"github.com/spf13/cobra"
)

// UpCmd returns nil on Windows as this command is not supported.
func UpCmd() *cobra.Command {
	return nil
}