package create

import (
	"fmt"

	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Create a new client",
	Run: func(c *cobra.Command, args []string) {
		fmt.Println("Client created successfully!")
	},
}

func init() {
	CreateCmd.AddCommand(clientCmd)
}
