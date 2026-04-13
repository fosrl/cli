package list

import "github.com/spf13/cobra"

// ListCmd is the parent `list` command for listing server-side items.
func ListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List resources and other items from the server",
	}
	cmd.AddCommand(aliasesCmd())
	return cmd
}
