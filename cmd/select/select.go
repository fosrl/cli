package selectcmd

import (
	"github.com/fosrl/cli/cmd/select/account"
	"github.com/fosrl/cli/cmd/select/exitnode"
	"github.com/fosrl/cli/cmd/select/org"
	"github.com/spf13/cobra"
)

func SelectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "select",
		Short: "Select account information to use",
		Long:  "Select account information to use",
	}

	cmd.AddCommand(account.AccountCmd())
	cmd.AddCommand(org.OrgCmd())
	cmd.AddCommand(exitnode.ExitNodeCmd())

	return cmd
}
