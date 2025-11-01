package list

import (
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources",
	Long:  "List resources such as sites, clients, etc.",
}

