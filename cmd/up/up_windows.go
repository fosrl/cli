//go:build windows

package up

import (
	"github.com/fosrl/cli/cmd/up/site"
	"github.com/spf13/cobra"
)

// UpCmd returns the "up" command tree available on Windows. The `client`
// (Olm) subcommand is unix-only; `site` (Newt) supports Windows too.
func UpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start a connection",
		Long:  `Bring up a connection.`,
	}

	cmd.AddCommand(site.SiteUpCmd())

	return cmd
}