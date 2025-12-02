package status

import (
	"github.com/fosrl/cli/cmd/status/client"
	"github.com/spf13/cobra"
)

var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status commands",
	Long:  "View status information",
	Run: func(cmd *cobra.Command, args []string) {
		// Default to client subcommand if no subcommand is provided
		// This makes "pangolin status" equivalent to "pangolin status client"
		client.ClientCmd.Run(client.ClientCmd, args)
	},
}

func init() {
	StatusCmd.AddCommand(client.ClientCmd)
}

