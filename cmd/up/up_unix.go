//go:build !windows

package up

import (
	"github.com/fosrl/cli/cmd/up/client"
	"github.com/fosrl/cli/cmd/up/site"
	"github.com/spf13/cobra"
)

func UpCmd() *cobra.Command {
	// If no subcommand is specified, run the `client`
	// subcommand by default.
	cmd := client.ClientUpCmd()

	cmd.Use = "up"
	cmd.Short = "Start a connection"
	cmd.Long = `Bring up a connection.

If ran with no subcommand, 'client' is passed.
`

	cmd.AddCommand(client.ClientUpCmd())
	cmd.AddCommand(site.SiteUpCmd())

	return cmd
}
