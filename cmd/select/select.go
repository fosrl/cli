package selectcmd

import (
	"github.com/spf13/cobra"
)

var SelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select organization",
	Long:  "Select an organization to work with",
}
