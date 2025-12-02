package down

import (
	"github.com/spf13/cobra"
)

var DownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop a client or site",
	Long:  "Stop a client or site tunneled connection",
	Run: func(cmd *cobra.Command, args []string) {
		// Default to client subcommand if no subcommand is provided
		// This makes "pangolin down" equivalent to "pangolin down client"
		ClientCmd.Run(ClientCmd, args)
	},
}
