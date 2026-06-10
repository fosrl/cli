//go:build windows

package resetdns

import "github.com/spf13/cobra"

// ResetDNSCmd is unsupported on Windows where DNS overrides are
// interface-GUID scoped and reclaimed automatically when the WireGuard
// interface is torn down.
func ResetDNSCmd() *cobra.Command {
	return nil
}
