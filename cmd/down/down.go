package down

import (
	"github.com/fosrl/cli/cmd/down/client"
	"github.com/spf13/cobra"
)

func DownCmd() *cobra.Command {
	// If no subcommand is specified, run the `client`
	// subcommand by default.
	cmd := client.ClientDownCmd()

	cmd.Use = "down"
	cmd.Short = "Stop a connection"
	cmd.Long = `Bring down a connection.

If ran with no subcommand, 'client' is passed.
`

	cmd.AddCommand(client.ClientDownCmd())

	return cmd
}
