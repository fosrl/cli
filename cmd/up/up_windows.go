//go:build windows

package up

import (
	"github.com/fosrl/cli/cmd/up/client"
	"github.com/fosrl/cli/cmd/up/site"
	"github.com/spf13/cobra"
)

// UpCmd returns the "up" command tree available on Windows. `site` (Newt)
// fully supports Windows; `client` (Olm) only supports machine clients
// there (see client.ClientUpCmd's Long text) - interactive login isn't
// implemented on Windows yet.
func UpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start a connection",
		Long:  `Bring up a connection.`,
	}

	cmd.AddCommand(client.ClientUpCmd())
	cmd.AddCommand(site.SiteUpCmd())

	return cmd
}
