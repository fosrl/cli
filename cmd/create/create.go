package create

import (
	"github.com/spf13/cobra"
)

var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create resources",
	Long:  "Create resources such as sites, clients, etc.",
}

