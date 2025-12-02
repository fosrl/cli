package up

import (
	"github.com/spf13/cobra"
)

var UpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start a client or site",
	Long:  "Bring up a client or site tunneled connection",
	Run: func(cmd *cobra.Command, args []string) {
		// Default to client subcommand if no subcommand is provided
		// This makes "pangolin up" equivalent to "pangolin up client"
		ClientCmd.Run(ClientCmd, args)
	},
}
