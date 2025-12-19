package status

import (
	"github.com/fosrl/cli/cmd/status/client"
	"github.com/spf13/cobra"
)

func StatusCmd() *cobra.Command {
	// If no subcommand is specified, run the `client`
	// subcommand by default.
	cmd := client.ClientStatusCmd()

	cmd.Use = "status"
	cmd.Short = "Status commands"
	cmd.Long = `View status information.

If ran with no subcommand, 'client' is passed.
`

	cmd.AddCommand(client.ClientStatusCmd())

	return cmd
}
