package selectcmd

import (
	"github.com/fosrl/cli/cmd/select/org"
	"github.com/spf13/cobra"
)

var SelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select objects to work with",
	Long:  "Select objects to work with",
}

func init() {
	SelectCmd.AddCommand(org.OrgCmd)
}
