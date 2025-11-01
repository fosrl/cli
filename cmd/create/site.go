package create

import (
	"fmt"

	"github.com/spf13/cobra"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Create a new site",
	Run: func(c *cobra.Command, args []string) {
		fmt.Println("Site created successfully!")
	},
}

func init() {
	CreateCmd.AddCommand(siteCmd)
}
