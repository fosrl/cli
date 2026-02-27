//go:build windows

package logs

import (
	"github.com/spf13/cobra"
)

// LogsCmd returns nil on Windows as this command is not supported.
func LogsCmd() *cobra.Command {
	return nil
}