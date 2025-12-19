package logs

import (
	"github.com/fosrl/cli/cmd/logs/client"
	"github.com/spf13/cobra"
)

func LogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View client logs",
		Long:  "View and follow client logs",
	}

	cmd.AddCommand(client.ClientLogsCmd())

	return cmd
}
