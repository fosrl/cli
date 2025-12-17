package up

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var UpCmd = &cobra.Command{
	Use:   "up",
	Short: "Start a client",
	Long:  "Bring up a client connection",
	Run: func(cmd *cobra.Command, args []string) {
		// Default to client subcommand if no subcommand is provided
		// This makes "pangolin up" equivalent to "pangolin up client"
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if cmd.Flags().Changed(flag.Name) {
				// Ensure stringSlice flags are passed without the bracketed representation
				if flag.Value.Type() == "stringSlice" {
					if vals, err := cmd.Flags().GetStringSlice(flag.Name); err == nil {
						ClientCmd.Flags().Set(flag.Name, strings.Join(vals, ","))
						return
					}
				}
				ClientCmd.Flags().Set(flag.Name, flag.Value.String())
			}
		})
		ClientCmd.Run(ClientCmd, args)
	},
}

func init() {
	addClientFlags(UpCmd)
}
